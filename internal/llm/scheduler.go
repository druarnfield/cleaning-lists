package llm

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/druarnfield/cleaning-scheduler/internal/scheduler"
)

// GenerateSchedule uses LLM to generate task instances for the specified number of weeks
func GenerateSchedule(ctx context.Context, db *sqlc.Queries, startDate time.Time, weeksAhead int) error {
	// Calculate next week's Monday (we generate from next week, not current week)
	currentWeekStart := scheduler.GetWeekStart(startDate)
	nextWeekStart := currentWeekStart.AddDate(0, 0, 7)
	endDate := nextWeekStart.AddDate(0, 0, weeksAhead*7)

	log.Printf("Starting LLM-powered schedule generation for %d weeks from %s", weeksAhead, nextWeekStart.Format("2006-01-02"))

	// Delete existing instances in the target date range BEFORE building context
	log.Printf("Clearing existing instances from %s to %s", nextWeekStart.Format("2006-01-02"), endDate.Format("2006-01-02"))
	err := db.DeleteInstancesInDateRange(ctx, sqlc.DeleteInstancesInDateRangeParams{
		WeekStartDate:   nextWeekStart,
		WeekStartDate_2: endDate,
	})
	if err != nil {
		log.Printf("Warning: failed to delete existing instances: %v", err)
	}

	// Build context for LLM using next week as the start (after clearing old instances)
	schedulingContext, err := BuildSchedulingContext(ctx, db, nextWeekStart, weeksAhead)
	if err != nil {
		return fmt.Errorf("failed to build scheduling context: %w", err)
	}

	// Create LLM client
	client, err := NewClient()
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Get prompts
	systemPrompt := GetSystemPrompt()
	userPrompt := GetSchedulingPrompt(schedulingContext, weeksAhead)

	log.Println("Sending scheduling request to Claude...")

	// Call LLM with retry
	response, err := client.SendMessageWithRetry(ctx, systemPrompt, userPrompt, 8192, 3)
	if err != nil {
		return fmt.Errorf("failed to get LLM response: %w", err)
	}

	// Parse and validate response
	scheduleResponse, err := ParseLLMResponse(response)
	if err != nil {
		return fmt.Errorf("failed to parse LLM response: %w", err)
	}

	validationErrors := ValidateScheduleResponse(scheduleResponse, schedulingContext.Tasks, schedulingContext.DateRange)
	if len(validationErrors) > 0 {
		errMsgs := []string{}
		for _, e := range validationErrors {
			errMsgs = append(errMsgs, e.Error())
		}
		return fmt.Errorf("LLM response validation failed:\n%s", strings.Join(errMsgs, "\n"))
	}

	log.Printf("LLM generated %d task instances. Summary: %s", len(scheduleResponse.Instances), scheduleResponse.Summary)

	// Create task instances from LLM response
	for _, instance := range scheduleResponse.Instances {
		// Parse dates from strings
		scheduledDate, err := time.Parse("2006-01-02", instance.ScheduledDate)
		if err != nil {
			return fmt.Errorf("failed to parse scheduled date '%s': %w", instance.ScheduledDate, err)
		}
		weekStartDate, err := time.Parse("2006-01-02", instance.WeekStartDate)
		if err != nil {
			return fmt.Errorf("failed to parse week start date '%s': %w", instance.WeekStartDate, err)
		}

		_, err = db.CreateInstance(ctx, sqlc.CreateInstanceParams{
			TaskID:        instance.TaskID,
			ScheduledDate: scheduledDate,
			OriginalScheduledDate: sql.NullTime{
				Valid: false,
			},
			AssignedTo:    instance.AssignedTo,
			WeekStartDate: weekStartDate,
			BroughtForward: sql.NullBool{
				Bool:  false,
				Valid: true,
			},
			BroughtForwardBy: sql.NullString{
				Valid: false,
			},
			CountsTowardWeekly: sql.NullBool{
				Bool:  true,
				Valid: true,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create task instance for task %d on %s: %w",
				instance.TaskID, instance.ScheduledDate, err)
		}
	}

	log.Printf("Successfully created %d task instances", len(scheduleResponse.Instances))

	// Handle task changes if requested
	if len(scheduleResponse.TaskChanges) > 0 {
		log.Printf("Processing %d task changes...", len(scheduleResponse.TaskChanges))
		for _, change := range scheduleResponse.TaskChanges {
			if err := applyTaskChange(ctx, db, change); err != nil {
				log.Printf("Warning: failed to apply task change: %v", err)
			}
		}
	}

	return nil
}

// AdjustSchedule uses LLM to adjust the schedule based on a user request
func AdjustSchedule(ctx context.Context, db *sqlc.Queries, userMessage string, weekOffset int) (string, bool, error) {
	log.Printf("Adjusting schedule based on user request (week offset %d): %s", weekOffset, userMessage)

	// Calculate current week and next week (where future schedule starts)
	now := time.Now()
	currentWeekStart := scheduler.GetWeekStart(now)
	nextWeekStart := currentWeekStart.AddDate(0, 0, 7)

	// Get the configured weeks ahead to determine the full schedule range
	weeksAhead := ParseWeeksAhead()
	endDate := nextWeekStart.AddDate(0, 0, weeksAhead*7)

	// Build context for all future weeks - SHOW LLM the current schedule so it can make adjustments
	schedulingContext, err := BuildSchedulingContext(ctx, db, nextWeekStart, weeksAhead)
	if err != nil {
		return "", false, fmt.Errorf("failed to build scheduling context: %w", err)
	}

	// Filter out completed instances from context (LLM shouldn't modify these)
	filteredInstances := []InstanceInfo{}
	for _, instance := range schedulingContext.ExistingInstances {
		_, err := db.GetCompletionByInstance(ctx, instance.ID)
		if err == sql.ErrNoRows {
			// Not completed, include it for LLM to see and potentially adjust
			filteredInstances = append(filteredInstances, instance)
		}
		// Skip completed instances
	}
	schedulingContext.ExistingInstances = filteredInstances

	// Create LLM client
	client, err := NewClient()
	if err != nil {
		return "", false, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Get prompts
	systemPrompt := GetSystemPrompt()
	userPrompt := GetAdjustmentPrompt(schedulingContext, userMessage, weekOffset, weeksAhead)

	log.Println("Sending adjustment request to Claude...")

	// Call LLM with retry
	response, err := client.SendMessageWithRetry(ctx, systemPrompt, userPrompt, 8192, 3)
	if err != nil {
		return "", false, fmt.Errorf("failed to get LLM response: %w", err)
	}

	// Parse and validate response
	adjustmentResponse, err := ParseLLMResponse(response)
	if err != nil {
		return "", false, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	validationErrors := ValidateScheduleResponse(adjustmentResponse, schedulingContext.Tasks, schedulingContext.DateRange)
	if len(validationErrors) > 0 {
		errMsgs := []string{}
		for _, e := range validationErrors {
			errMsgs = append(errMsgs, e.Error())
		}
		return "", false, fmt.Errorf("LLM response validation failed:\n%s", strings.Join(errMsgs, "\n"))
	}

	// Check if any changes were made
	changedSomething := len(adjustmentResponse.Instances) > 0 || len(adjustmentResponse.TaskChanges) > 0

	if !changedSomething {
		return adjustmentResponse.Summary, false, nil
	}

	// Delete existing non-completed instances before creating adjusted schedule
	log.Printf("Applying adjustments - clearing non-completed instances")
	instancesToDelete, err := db.ListInstancesByDateRange(ctx, sqlc.ListInstancesByDateRangeParams{
		ScheduledDate:   nextWeekStart,
		ScheduledDate_2: endDate,
	})
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Warning: failed to get instances to delete: %v", err)
	}

	// Delete only non-completed instances
	deletedCount := 0
	for _, instance := range instancesToDelete {
		_, err := db.GetCompletionByInstance(ctx, instance.ID)
		if err == sql.ErrNoRows {
			// Not completed, delete it
			err = db.DeleteInstance(ctx, instance.ID)
			if err != nil {
				log.Printf("Warning: failed to delete instance %d: %v", instance.ID, err)
			} else {
				deletedCount++
			}
		}
	}
	log.Printf("Deleted %d non-completed instances", deletedCount)

	// Create new instances based on LLM adjustments
	for _, instance := range adjustmentResponse.Instances {
		// Parse dates from strings
		scheduledDate, err := time.Parse("2006-01-02", instance.ScheduledDate)
		if err != nil {
			return "", false, fmt.Errorf("failed to parse scheduled date '%s': %w", instance.ScheduledDate, err)
		}
		weekStartDate, err := time.Parse("2006-01-02", instance.WeekStartDate)
		if err != nil {
			return "", false, fmt.Errorf("failed to parse week start date '%s': %w", instance.WeekStartDate, err)
		}

		_, err = db.CreateInstance(ctx, sqlc.CreateInstanceParams{
			TaskID:        instance.TaskID,
			ScheduledDate: scheduledDate,
			OriginalScheduledDate: sql.NullTime{
				Valid: false,
			},
			AssignedTo:    instance.AssignedTo,
			WeekStartDate: weekStartDate,
			BroughtForward: sql.NullBool{
				Bool:  false,
				Valid: true,
			},
			BroughtForwardBy: sql.NullString{
				Valid: false,
			},
			CountsTowardWeekly: sql.NullBool{
				Bool:  true,
				Valid: true,
			},
		})
		if err != nil {
			return "", false, fmt.Errorf("failed to create task instance: %w", err)
		}
	}

	// Handle task changes if requested
	if len(adjustmentResponse.TaskChanges) > 0 {
		log.Printf("Processing %d task changes...", len(adjustmentResponse.TaskChanges))
		for _, change := range adjustmentResponse.TaskChanges {
			if err := applyTaskChange(ctx, db, change); err != nil {
				log.Printf("Warning: failed to apply task change: %v", err)
			}
		}
	}

	log.Printf("Schedule adjustment complete. %d instances created.", len(adjustmentResponse.Instances))

	return adjustmentResponse.Summary, true, nil
}

// applyTaskChange applies a task definition change (create/update/delete)
func applyTaskChange(ctx context.Context, db *sqlc.Queries, change TaskChangeRequest) error {
	switch change.Action {
	case "create":
		_, err := db.CreateTask(ctx, sqlc.CreateTaskParams{
			Name:          change.Name,
			Category:      change.Category,
			Frequency:     change.Frequency,
			EstimatedMins: change.EstimatedMins,
			DefaultAssignee: sql.NullString{
				String: func() string {
					if change.DefaultAssignee != nil {
						return *change.DefaultAssignee
					}
					return ""
				}(),
				Valid: change.DefaultAssignee != nil,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
		log.Printf("Created new task: %s", change.Name)

	case "update":
		if change.TaskID == nil {
			return fmt.Errorf("task_id is required for update")
		}
		_, err := db.UpdateTask(ctx, sqlc.UpdateTaskParams{
			ID:            *change.TaskID,
			Name:          change.Name,
			Category:      change.Category,
			Frequency:     change.Frequency,
			EstimatedMins: change.EstimatedMins,
			DefaultAssignee: sql.NullString{
				String: func() string {
					if change.DefaultAssignee != nil {
						return *change.DefaultAssignee
					}
					return ""
				}(),
				Valid: change.DefaultAssignee != nil,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to update task: %w", err)
		}
		log.Printf("Updated task %d: %s", *change.TaskID, change.Name)

	case "delete":
		if change.TaskID == nil {
			return fmt.Errorf("task_id is required for delete")
		}
		err := db.DeleteTask(ctx, *change.TaskID)
		if err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}
		log.Printf("Deleted task %d", *change.TaskID)

	default:
		return fmt.Errorf("unknown action: %s", change.Action)
	}

	return nil
}
