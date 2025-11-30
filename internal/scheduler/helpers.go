package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
)

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
		TaskID:                taskID,
		ScheduledDate:         toDate,
		OriginalScheduledDate: sql.NullTime{Time: nextInstance.ScheduledDate, Valid: true},
		AssignedTo:            byUser, // Assign to the person bringing it forward
		WeekStartDate:         weekStart,
		BroughtForward:        sql.NullBool{Bool: true, Valid: true},
		BroughtForwardBy:      sql.NullString{String: byUser, Valid: true},
		CountsTowardWeekly:    sql.NullBool{Bool: false, Valid: true}, // Doesn't count toward weekly totals
	})

	if err != nil {
		return fmt.Errorf("failed to create brought-forward instance: %w", err)
	}

	return nil
}

// GetWorkloadSummary calculates the total weekly workload based on task definitions
func GetWorkloadSummary(ctx context.Context, db *sqlc.Queries) (druMins, josieMins float64, err error) {
	tasks, err := db.ListTasks(ctx)
	if err != nil {
		return 0, 0, err
	}

	for _, task := range tasks {
		frequencyDays, err := ParseFrequency(task.Frequency)
		if err != nil {
			continue
		}
		weeklyMins := GetWeeklyMinutes(frequencyDays, int(task.EstimatedMins))

		// Calculate based on default assignee
		if !task.DefaultAssignee.Valid {
			// If no default, assume it's distributed (we can't determine without running the LLM)
			// For now, split it evenly
			druMins += weeklyMins / 2
			josieMins += weeklyMins / 2
			continue
		}

		assignee := strings.ToLower(task.DefaultAssignee.String)
		switch assignee {
		case "dru":
			druMins += weeklyMins
		case "josie":
			josieMins += weeklyMins
		case "both":
			// Split equally for "both" tasks
			druMins += weeklyMins / 2
			josieMins += weeklyMins / 2
		case "alternate":
			// For alternate, split evenly
			druMins += weeklyMins / 2
			josieMins += weeklyMins / 2
		}
	}

	return druMins, josieMins, nil
}
