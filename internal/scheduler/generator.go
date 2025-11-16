package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
)

// GetWeekStart returns the Monday of the week containing the given date
func GetWeekStart(date time.Time) time.Time {
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday should be 7 for this calculation
	}
	daysFromMonday := weekday - 1
	monday := date.AddDate(0, 0, -daysFromMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// GenerateInstances generates task instances for the specified number of weeks
func GenerateInstances(ctx context.Context, db *sqlc.Queries, startDate time.Time, weeksAhead int) error {
	// Get task assignments
	assignments, err := DistributeTasks(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to distribute tasks: %w", err)
	}

	// Get all tasks
	tasks, err := db.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	endDate := startDate.AddDate(0, 0, weeksAhead*7)

	// Process each task
	for _, task := range tasks {
		// Parse frequency
		frequencyDays, err := ParseFrequency(task.Frequency)
		if err != nil {
			return fmt.Errorf("failed to parse frequency for task %s: %w", task.Name, err)
		}

		assignee := assignments[task.ID]

		// Handle "both" tasks by generating instances for both people
		if assignee == "both" {
			// Generate for Dru
			err = generateInstancesForPerson(ctx, db, task, "dru", startDate, endDate, frequencyDays)
			if err != nil {
				return err
			}
			// Generate for Josie
			err = generateInstancesForPerson(ctx, db, task, "josie", startDate, endDate, frequencyDays)
			if err != nil {
				return err
			}
		} else {
			// Generate for single assignee
			err = generateInstancesForPerson(ctx, db, task, assignee, startDate, endDate, frequencyDays)
			if err != nil {
				return err
			}
		}
	}

	// Balance task instances across weeks to ensure fair weekly workloads
	err = BalanceWeeklyInstances(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to balance weekly instances: %w", err)
	}

	return nil
}

func generateInstancesForPerson(
	ctx context.Context,
	db *sqlc.Queries,
	task sqlc.Task,
	assignee string,
	startDate, endDate time.Time,
	frequencyDays float64,
) error {
	// Calculate first occurrence - spread tasks evenly without randomization
	// Use a small offset based on task ID to avoid all tasks on the same day
	dayOffset := float64(task.ID % int64(frequencyDays))
	if dayOffset >= frequencyDays {
		dayOffset = 0
	}
	firstOccurrence := startDate.Add(time.Duration(dayOffset*24) * time.Hour)

	// Generate recurring instances
	nextDate := firstOccurrence
	for nextDate.Before(endDate) || nextDate.Equal(endDate) {
		// Check if instance already exists (to avoid duplicates)
		count, err := db.CheckDuplicateInstance(ctx, sqlc.CheckDuplicateInstanceParams{
			TaskID:        task.ID,
			ScheduledDate: nextDate,
		})
		if err != nil {
			return fmt.Errorf("failed to check duplicate instance: %w", err)
		}

		if count == 0 {
			// Create instance
			weekStart := GetWeekStart(nextDate)
			_, err = db.CreateInstance(ctx, sqlc.CreateInstanceParams{
				TaskID:               task.ID,
				ScheduledDate:        nextDate,
				OriginalScheduledDate: sql.NullTime{Valid: false},
				AssignedTo:           assignee,
				WeekStartDate:        weekStart,
				BroughtForward:       sql.NullBool{Bool: false, Valid: true},
				BroughtForwardBy:     sql.NullString{Valid: false},
				CountsTowardWeekly:   sql.NullBool{Bool: true, Valid: true},
			})
			if err != nil {
				return fmt.Errorf("failed to create instance for task %s: %w", task.Name, err)
			}
		}

		// Calculate next occurrence
		nextDate = nextDate.Add(time.Duration(frequencyDays*24) * time.Hour)
	}

	return nil
}

// RegenerateAllInstances clears and regenerates all instances
func RegenerateAllInstances(ctx context.Context, db *sqlc.Queries) error {
	// Delete all existing instances (but preserve completions through foreign key)
	// This would need a new query in sqlc
	// For now, we'll just generate new instances without deleting

	startDate := time.Now().Truncate(24 * time.Hour)
	return GenerateInstances(ctx, db, startDate, 4)
}

// BringForwardTask creates a brought-forward instance of a task
func BringForwardTask(ctx context.Context, db *sqlc.Queries, taskID int64, toDate time.Time, byUser string) error {
	// Normalize username
	byUser = strings.ToLower(strings.TrimSpace(byUser))

	// Get the next future instance of this task
	nextInstance, err := db.GetNextInstanceForTask(ctx, sqlc.GetNextInstanceForTaskParams{
		TaskID:        taskID,
		ScheduledDate: time.Now(),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no future instances found for this task")
		}
		return fmt.Errorf("failed to get next instance: %w", err)
	}

	// Check if the user can bring forward this task
	// They can if it's assigned to them or if it's a "both" task
	task, err := db.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	canBringForward := false
	if nextInstance.AssignedTo == byUser {
		canBringForward = true
	} else if task.DefaultAssignee.Valid && strings.ToLower(task.DefaultAssignee.String) == "both" {
		canBringForward = true
	}

	if !canBringForward {
		return fmt.Errorf("you can only bring forward tasks assigned to you")
	}

	// Create the brought-forward instance
	weekStart := GetWeekStart(toDate)
	_, err = db.CreateInstance(ctx, sqlc.CreateInstanceParams{
		TaskID:               taskID,
		ScheduledDate:        toDate,
		OriginalScheduledDate: sql.NullTime{Time: nextInstance.ScheduledDate, Valid: true},
		AssignedTo:           byUser, // Assign to the person bringing it forward
		WeekStartDate:        weekStart,
		BroughtForward:       sql.NullBool{Bool: true, Valid: true},
		BroughtForwardBy:     sql.NullString{String: byUser, Valid: true},
		CountsTowardWeekly:   sql.NullBool{Bool: false, Valid: true}, // Doesn't count toward weekly totals
	})

	if err != nil {
		return fmt.Errorf("failed to create brought-forward instance: %w", err)
	}

	return nil
}