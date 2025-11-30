package llm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/druarnfield/cleaning-scheduler/internal/scheduler"
)

// SchedulingContext contains all the information needed for LLM scheduling
type SchedulingContext struct {
	Tasks              []TaskInfo
	ExistingInstances  []InstanceInfo
	RecentCompletions  []CompletionInfo
	DateRange          DateRange
	CurrentWorkload    WorkloadInfo
}

type TaskInfo struct {
	ID              int64
	Name            string
	Category        string
	Frequency       string
	FrequencyDays   float64
	EstimatedMins   int64
	DefaultAssignee sql.NullString
	WeeklyMinutes   float64
	NextDueDate     *time.Time
	DaysUntilDue    int
}

type InstanceInfo struct {
	ID               int64
	TaskID           int64
	TaskName         string
	ScheduledDate    string
	AssignedTo       string
	WeekStartDate    string
	BroughtForward   bool
	CountsTowardWeekly bool
}

type CompletionInfo struct {
	TaskName    string
	AssignedTo  string
	CompletedBy string
	CompletedAt time.Time
}

type DateRange struct {
	StartDate string
	EndDate   string
	WeeksAhead int
}

type WorkloadInfo struct {
	DruMinutes   int64
	JosieMinutes int64
	DruTasks     int
	JosieTasks   int
}

// BuildSchedulingContext gathers all necessary context for the LLM
func BuildSchedulingContext(ctx context.Context, db *sqlc.Queries, startDate time.Time, weeksAhead int) (SchedulingContext, error) {
	weekStart := scheduler.GetWeekStart(startDate)
	endDate := weekStart.AddDate(0, 0, weeksAhead*7)

	context := SchedulingContext{
		DateRange: DateRange{
			StartDate:  weekStart.Format("2006-01-02"),
			EndDate:    endDate.Format("2006-01-02"),
			WeeksAhead: weeksAhead,
		},
	}

	// Get all tasks
	tasks, err := db.ListTasks(ctx)
	if err != nil {
		return context, fmt.Errorf("failed to get tasks: %w", err)
	}

	now := time.Now()
	for _, task := range tasks {
		freqDays, err := scheduler.ParseFrequency(task.Frequency)
		if err != nil {
			return context, fmt.Errorf("failed to parse frequency for task %s: %w", task.Name, err)
		}

		weeklyMins := scheduler.GetWeeklyMinutes(freqDays, int(task.EstimatedMins))

		taskInfo := TaskInfo{
			ID:              task.ID,
			Name:            task.Name,
			Category:        task.Category,
			Frequency:       task.Frequency,
			FrequencyDays:   freqDays,
			EstimatedMins:   task.EstimatedMins,
			DefaultAssignee: task.DefaultAssignee,
			WeeklyMinutes:   weeklyMins,
		}

		// Calculate next due date from task creation date
		if task.CreatedAt.Valid {
			createdAt := task.CreatedAt.Time
			freqDaysInt := int(freqDays)

			// Calculate how many periods have passed since creation
			daysSinceCreation := int(now.Sub(createdAt).Hours() / 24)
			periodsPassed := daysSinceCreation / freqDaysInt

			// Next due is: creation + (periods_passed + 1) * frequency
			nextDue := createdAt.AddDate(0, 0, (periodsPassed+1)*freqDaysInt)
			taskInfo.NextDueDate = &nextDue
			taskInfo.DaysUntilDue = int(nextDue.Sub(now).Hours() / 24)
		}

		context.Tasks = append(context.Tasks, taskInfo)
	}

	// Get existing instances in the target date range (for adjustment scenarios)
	instances, err := db.ListInstancesByDateRange(ctx, sqlc.ListInstancesByDateRangeParams{
		ScheduledDate:   weekStart,
		ScheduledDate_2: endDate,
	})
	if err != nil && err != sql.ErrNoRows {
		return context, fmt.Errorf("failed to get existing instances: %w", err)
	}

	for _, instance := range instances {
		context.ExistingInstances = append(context.ExistingInstances, InstanceInfo{
			ID:                 instance.ID,
			TaskID:             instance.TaskID,
			TaskName:           instance.Name,
			ScheduledDate:      instance.ScheduledDate.Format("2006-01-02"),
			AssignedTo:         instance.AssignedTo,
			WeekStartDate:      instance.WeekStartDate.Format("2006-01-02"),
			BroughtForward:     instance.BroughtForward.Bool,
			CountsTowardWeekly: instance.CountsTowardWeekly.Bool,
		})
	}

	// Get recent completions (last 4 weeks) for historical context
	historicalStartDate := weekStart.AddDate(0, 0, -28)
	completions, err := db.ListCompletionsByDateRange(ctx, sqlc.ListCompletionsByDateRangeParams{
		ScheduledDate:   historicalStartDate,
		ScheduledDate_2: weekStart,
	})
	if err != nil && err != sql.ErrNoRows {
		return context, fmt.Errorf("failed to get completions: %w", err)
	}

	for _, completion := range completions {
		context.RecentCompletions = append(context.RecentCompletions, CompletionInfo{
			TaskName:    completion.Name,
			AssignedTo:  completion.AssignedTo,
			CompletedBy: completion.CompletedBy,
			CompletedAt: completion.CompletedAt.Time,
		})
	}

	// Calculate current workload for the target period
	druMins := int64(0)
	josieMins := int64(0)
	druTasks := 0
	josieTasks := 0

	for _, instance := range context.ExistingInstances {
		if !instance.CountsTowardWeekly {
			continue
		}

		// Find the task to get estimated minutes
		for _, task := range context.Tasks {
			if task.ID == instance.TaskID {
				if instance.AssignedTo == "dru" {
					druMins += task.EstimatedMins
					druTasks++
				} else if instance.AssignedTo == "josie" {
					josieMins += task.EstimatedMins
					josieTasks++
				}
				break
			}
		}
	}

	context.CurrentWorkload = WorkloadInfo{
		DruMinutes:   druMins,
		JosieMinutes: josieMins,
		DruTasks:     druTasks,
		JosieTasks:   josieTasks,
	}

	return context, nil
}

// Format converts the context into a formatted string for the LLM prompt
func (c *SchedulingContext) Format() string {
	var sb strings.Builder

	sb.WriteString("DATE RANGE: ")
	sb.WriteString(fmt.Sprintf("%s to %s (%d weeks)\n\n", c.DateRange.StartDate, c.DateRange.EndDate, c.DateRange.WeeksAhead))

	sb.WriteString("TASKS:\n")
	for _, task := range c.Tasks {
		assignee := "null"
		if task.DefaultAssignee.Valid {
			assignee = task.DefaultAssignee.String
		}
		dueInfo := ""
		if task.NextDueDate != nil {
			dueInfo = fmt.Sprintf(" | Next due: %s (%d days)", task.NextDueDate.Format("2006-01-02"), task.DaysUntilDue)
		}
		sb.WriteString(fmt.Sprintf("ID:%d | %s | %s | %s (%.0f days) | %dmin/task (%.1fmin/week) | default_assignee:%s%s\n",
			task.ID, task.Name, task.Category, task.Frequency, task.FrequencyDays, task.EstimatedMins, task.WeeklyMinutes, assignee, dueInfo))
	}
	sb.WriteString("\n")

	if len(c.ExistingInstances) > 0 {
		sb.WriteString("EXISTING INSTANCES (non-completed):\n")
		for _, instance := range c.ExistingInstances {
			sb.WriteString(fmt.Sprintf("ID:%d | Task:%d | %s | Date:%s | Assigned:%s\n",
				instance.ID, instance.TaskID, instance.TaskName, instance.ScheduledDate, instance.AssignedTo))
		}
		sb.WriteString("\n")
	}

	if len(c.RecentCompletions) > 0 {
		sb.WriteString("RECENT COMPLETIONS (last 4 weeks):\n")
		completionCounts := make(map[string]map[string]int)
		for _, comp := range c.RecentCompletions {
			if completionCounts[comp.TaskName] == nil {
				completionCounts[comp.TaskName] = make(map[string]int)
			}
			completionCounts[comp.TaskName][comp.CompletedBy]++
		}
		for taskName, counts := range completionCounts {
			sb.WriteString(fmt.Sprintf("%s: ", taskName))
			parts := []string{}
			for assignee, count := range counts {
				parts = append(parts, fmt.Sprintf("%s=%d", assignee, count))
			}
			sb.WriteString(strings.Join(parts, ", "))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if c.CurrentWorkload.DruMinutes > 0 || c.CurrentWorkload.JosieMinutes > 0 {
		sb.WriteString("CURRENT WORKLOAD:\n")
		sb.WriteString(fmt.Sprintf("Dru: %dmin (%d tasks) | Josie: %dmin (%d tasks) | Diff: %dmin\n\n",
			c.CurrentWorkload.DruMinutes, c.CurrentWorkload.DruTasks,
			c.CurrentWorkload.JosieMinutes, c.CurrentWorkload.JosieTasks,
			c.CurrentWorkload.DruMinutes-c.CurrentWorkload.JosieMinutes))
	}

	return sb.String()
}
