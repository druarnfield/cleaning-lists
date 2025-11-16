package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
)

type TaskAssignment struct {
	TaskID     int64
	TaskName   string
	AssignedTo string
	WeeklyMins float64
}

type TaskForDistribution struct {
	ID           int64
	Name         string
	WeeklyMins   float64
	DefaultAssignee sql.NullString
}

// DistributeTasks fairly distributes tasks between Dru and Josie
// Returns a map of task IDs to assigned person
func DistributeTasks(ctx context.Context, db *sqlc.Queries) (map[int64]string, error) {
	// Get all tasks
	tasks, err := db.ListTasks(ctx)
	if err != nil {
		return nil, err
	}

	assignments := make(map[int64]string)
	var druTotal, josieTotal float64
	var unassignedTasks []TaskForDistribution

	// Process each task
	for _, task := range tasks {
		// Calculate weekly minutes for this task
		frequencyDays, err := ParseFrequency(task.Frequency)
		if err != nil {
			return nil, fmt.Errorf("error parsing frequency for task %s: %w", task.Name, err)
		}
		weeklyMins := GetWeeklyMinutes(frequencyDays, int(task.EstimatedMins))

		// Handle based on default assignee
		if task.DefaultAssignee.Valid {
			assignee := strings.ToLower(task.DefaultAssignee.String)

			switch assignee {
			case "dru":
				assignments[task.ID] = "dru"
				druTotal += weeklyMins
			case "josie":
				assignments[task.ID] = "josie"
				josieTotal += weeklyMins
			case "both":
				// "Both" tasks get assigned to both people
				// We'll handle this by creating duplicate instances later
				assignments[task.ID] = "both"
				// Count half the time for each person
				druTotal += weeklyMins / 2
				josieTotal += weeklyMins / 2
			case "alternate":
				// Add to unassigned list for distribution
				unassignedTasks = append(unassignedTasks, TaskForDistribution{
					ID:           task.ID,
					Name:         task.Name,
					WeeklyMins:   weeklyMins,
					DefaultAssignee: task.DefaultAssignee,
				})
			default:
				// Unknown assignee, treat as unassigned
				unassignedTasks = append(unassignedTasks, TaskForDistribution{
					ID:           task.ID,
					Name:         task.Name,
					WeeklyMins:   weeklyMins,
					DefaultAssignee: task.DefaultAssignee,
				})
			}
		} else {
			// No assignee, add to unassigned list
			unassignedTasks = append(unassignedTasks, TaskForDistribution{
				ID:         task.ID,
				Name:       task.Name,
				WeeklyMins: weeklyMins,
				DefaultAssignee: task.DefaultAssignee,
			})
		}
	}

	// Sort unassigned tasks by weekly minutes (descending) for greedy algorithm
	sort.Slice(unassignedTasks, func(i, j int) bool {
		return unassignedTasks[i].WeeklyMins > unassignedTasks[j].WeeklyMins
	})

	// Distribute unassigned tasks using greedy algorithm
	for _, task := range unassignedTasks {
		if druTotal <= josieTotal {
			assignments[task.ID] = "dru"
			druTotal += task.WeeklyMins
		} else {
			assignments[task.ID] = "josie"
			josieTotal += task.WeeklyMins
		}
	}

	return assignments, nil
}

// GetWorkloadSummary calculates the total weekly workload for each person
func GetWorkloadSummary(ctx context.Context, db *sqlc.Queries) (druMins, josieMins float64, err error) {
	assignments, err := DistributeTasks(ctx, db)
	if err != nil {
		return 0, 0, err
	}

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

		assignee := assignments[task.ID]
		switch assignee {
		case "dru":
			druMins += weeklyMins
		case "josie":
			josieMins += weeklyMins
		case "both":
			// Split equally for "both" tasks
			druMins += weeklyMins / 2
			josieMins += weeklyMins / 2
		}
	}

	return druMins, josieMins, nil
}