package llm

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LLMScheduleResponse represents the expected JSON response from the LLM
type LLMScheduleResponse struct {
	Instances   []TaskInstanceRequest `json:"instances"`
	TaskChanges []TaskChangeRequest   `json:"task_changes,omitempty"`
	Summary     string                `json:"summary"`
}

// TaskInstanceRequest represents a single task instance to create
type TaskInstanceRequest struct {
	TaskID        int64  `json:"task_id"`
	ScheduledDate string `json:"scheduled_date"`
	AssignedTo    string `json:"assigned_to"`
	WeekStartDate string `json:"week_start_date"`
	Reasoning     string `json:"reasoning"`
}

// TaskChangeRequest represents a requested change to task definitions
type TaskChangeRequest struct {
	Action          string  `json:"action"` // create, update, delete
	TaskID          *int64  `json:"task_id"`
	Name            string  `json:"name"`
	Category        string  `json:"category"`
	Frequency       string  `json:"frequency"`
	EstimatedMins   int64   `json:"estimated_mins"`
	DefaultAssignee *string `json:"default_assignee"`
}

// ParseLLMResponse parses and validates the LLM's JSON response
func ParseLLMResponse(response string) (*LLMScheduleResponse, error) {
	// Try to extract JSON if the response has extra text
	response = strings.TrimSpace(response)

	// Find JSON object boundaries
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	var result LLMScheduleResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w\nResponse: %s", err, jsonStr)
	}

	return &result, nil
}

// ValidateScheduleResponse validates the LLM's schedule response
func ValidateScheduleResponse(response *LLMScheduleResponse, contextTasks []TaskInfo, dateRange DateRange) []error {
	var errors []error

	// Validate each instance
	taskMap := make(map[int64]TaskInfo)
	for _, task := range contextTasks {
		taskMap[task.ID] = task
	}

	startDate, err := time.Parse("2006-01-02", dateRange.StartDate)
	if err != nil {
		errors = append(errors, fmt.Errorf("invalid start date in context: %w", err))
	}

	endDate, err := time.Parse("2006-01-02", dateRange.EndDate)
	if err != nil {
		errors = append(errors, fmt.Errorf("invalid end date in context: %w", err))
	}

	for i := range response.Instances {
		instance := &response.Instances[i]
		// Validate task_id exists
		task, exists := taskMap[instance.TaskID]
		if !exists {
			errors = append(errors, fmt.Errorf("instance %d: task_id %d does not exist", i, instance.TaskID))
			continue
		}

		// Validate assigned_to
		if instance.AssignedTo != "dru" && instance.AssignedTo != "josie" {
			errors = append(errors, fmt.Errorf("instance %d: assigned_to must be 'dru' or 'josie', got '%s'", i, instance.AssignedTo))
		}

		// Validate scheduled_date format and range
		scheduledDate, err := time.Parse("2006-01-02", instance.ScheduledDate)
		if err != nil {
			errors = append(errors, fmt.Errorf("instance %d: invalid scheduled_date format '%s': %w", i, instance.ScheduledDate, err))
		} else {
			if scheduledDate.Before(startDate) || scheduledDate.After(endDate) {
				errors = append(errors, fmt.Errorf("instance %d: scheduled_date '%s' is outside the range %s to %s",
					i, instance.ScheduledDate, dateRange.StartDate, dateRange.EndDate))
			}
		}

		// Auto-calculate week_start_date if missing or empty
		if instance.WeekStartDate == "" {
			// Calculate from scheduled_date
			weekStart := scheduledDate
			// Find the Monday of the week containing scheduledDate
			for weekStart.Weekday() != time.Monday {
				weekStart = weekStart.AddDate(0, 0, -1)
			}
			instance.WeekStartDate = weekStart.Format("2006-01-02")
		} else {
			// Validate week_start_date format and is a Monday
			weekStart, err := time.Parse("2006-01-02", instance.WeekStartDate)
			if err != nil {
				errors = append(errors, fmt.Errorf("instance %d: invalid week_start_date format '%s': %w", i, instance.WeekStartDate, err))
			} else {
				if weekStart.Weekday() != time.Monday {
					errors = append(errors, fmt.Errorf("instance %d: week_start_date '%s' is not a Monday", i, instance.WeekStartDate))
				}
			}
		}

		// Validate pre-assigned tasks are respected
		if task.DefaultAssignee.Valid {
			assignee := strings.ToLower(task.DefaultAssignee.String)
			if assignee == "dru" && instance.AssignedTo != "dru" {
				errors = append(errors, fmt.Errorf("instance %d: task '%s' is pre-assigned to dru, but assigned to '%s'",
					i, task.Name, instance.AssignedTo))
			}
			if assignee == "josie" && instance.AssignedTo != "josie" {
				errors = append(errors, fmt.Errorf("instance %d: task '%s' is pre-assigned to josie, but assigned to '%s'",
					i, task.Name, instance.AssignedTo))
			}
			// Note: "both" and "alternate" are flexible and don't need validation
		}
	}

	// Validate task changes if present
	for i, change := range response.TaskChanges {
		if change.Action != "create" && change.Action != "update" && change.Action != "delete" {
			errors = append(errors, fmt.Errorf("task_change %d: action must be 'create', 'update', or 'delete', got '%s'", i, change.Action))
		}

		if change.Action == "create" && change.TaskID != nil {
			errors = append(errors, fmt.Errorf("task_change %d: task_id must be null for 'create' action", i))
		}

		if (change.Action == "update" || change.Action == "delete") && change.TaskID == nil {
			errors = append(errors, fmt.Errorf("task_change %d: task_id is required for '%s' action", i, change.Action))
		}

		if change.Action == "create" || change.Action == "update" {
			if change.Name == "" {
				errors = append(errors, fmt.Errorf("task_change %d: name is required", i))
			}
			if change.EstimatedMins <= 0 {
				errors = append(errors, fmt.Errorf("task_change %d: estimated_mins must be positive", i))
			}
		}
	}

	return errors
}
