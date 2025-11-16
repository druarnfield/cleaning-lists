package scheduler

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
)

// WeekWorkload represents the workload for a specific week
type WeekWorkload struct {
	Week       time.Time
	DruMins    float64
	JosieMins  float64
	Difference float64 // Absolute difference
	Percent    float64 // Percentage difference from average
}

// InstanceWithTask represents a task instance with its associated task details
type InstanceWithTask struct {
	Instance      sqlc.TaskInstance
	Task          sqlc.Task
	EstimatedMins int64
	CanSwap       bool // True if task can be swapped (not pre-assigned)
}

const (
	// BalanceThreshold is the maximum acceptable difference in minutes per week
	BalanceThreshold = 30.0
	// MaxSwapIterations prevents infinite loops
	MaxSwapIterations = 100
)

// BalanceWeeklyInstances balances task instances across weeks to achieve fair weekly workloads
func BalanceWeeklyInstances(ctx context.Context, db *sqlc.Queries) error {
	// Get all task instances
	instances, err := db.ListTaskInstances(ctx)
	if err != nil {
		return fmt.Errorf("failed to list task instances: %w", err)
	}

	// Get all tasks to access estimated minutes and default assignee
	tasks, err := db.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Create task lookup map
	taskMap := make(map[int64]sqlc.Task)
	for _, task := range tasks {
		taskMap[task.ID] = task
	}

	// Build instances with task details (only future instances)
	now := time.Now()
	instancesWithTasks := make([]InstanceWithTask, 0, len(instances))
	for _, instance := range instances {
		// Skip past instances
		if instance.WeekStartDate.Before(now) {
			continue
		}

		task, ok := taskMap[instance.TaskID]
		if !ok {
			continue
		}

		// Determine if this instance can be swapped
		// Only "Alternate" or NULL default assignees can be swapped
		// Case-insensitive check for "alternate"
		assignee := ""
		if task.DefaultAssignee.Valid {
			assignee = strings.ToLower(strings.TrimSpace(task.DefaultAssignee.String))
		}
		canSwap := !task.DefaultAssignee.Valid ||
			assignee == "" ||
			assignee == "alternate"

		instancesWithTasks = append(instancesWithTasks, InstanceWithTask{
			Instance:      instance,
			Task:          task,
			EstimatedMins: task.EstimatedMins,
			CanSwap:       canSwap,
		})
	}

	// Iteratively balance weeks
	swapsMade := 0
	for i := 0; i < MaxSwapIterations; i++ {
		// Calculate current workload per week
		weekWorkloads := calculateWeeklyWorkload(instancesWithTasks)

		// Find the most imbalanced week
		mostImbalanced := findMostImbalancedWeek(weekWorkloads)
		if mostImbalanced == nil || mostImbalanced.Difference <= BalanceThreshold {
			// All weeks are balanced, we're done!
			break
		}

		// Find the best swap for this week
		swapFound, err := findAndPerformBestSwap(ctx, db, instancesWithTasks, mostImbalanced.Week)
		if err != nil {
			return fmt.Errorf("failed to perform swap: %w", err)
		}

		if !swapFound {
			// No beneficial swap found for this week, try next iteration
			// This can happen if all tasks in the week are pre-assigned
			break
		}

		swapsMade++
	}

	return nil
}

// calculateWeeklyWorkload calculates the total minutes per person per week
func calculateWeeklyWorkload(instances []InstanceWithTask) map[time.Time]*WeekWorkload {
	workloads := make(map[time.Time]*WeekWorkload)

	for _, inst := range instances {
		week := inst.Instance.WeekStartDate
		if _, exists := workloads[week]; !exists {
			workloads[week] = &WeekWorkload{
				Week:      week,
				DruMins:   0,
				JosieMins: 0,
			}
		}

		if inst.Instance.AssignedTo == "dru" {
			workloads[week].DruMins += float64(inst.EstimatedMins)
		} else if inst.Instance.AssignedTo == "josie" {
			workloads[week].JosieMins += float64(inst.EstimatedMins)
		}
	}

	// Calculate differences and percentages
	for _, wl := range workloads {
		wl.Difference = math.Abs(wl.DruMins - wl.JosieMins)
		total := wl.DruMins + wl.JosieMins
		if total > 0 {
			wl.Percent = (wl.Difference / total) * 100
		}
	}

	return workloads
}

// findMostImbalancedWeek finds the week with the largest imbalance
func findMostImbalancedWeek(workloads map[time.Time]*WeekWorkload) *WeekWorkload {
	var mostImbalanced *WeekWorkload
	maxDifference := 0.0

	for _, wl := range workloads {
		if wl.Difference > maxDifference {
			maxDifference = wl.Difference
			mostImbalanced = wl
		}
	}

	return mostImbalanced
}

// findAndPerformBestSwap finds and executes the best task swap for a given week
func findAndPerformBestSwap(ctx context.Context, db *sqlc.Queries, instances []InstanceWithTask, week time.Time) (bool, error) {
	// Get ALL instances for this week to calculate true totals
	// Get swappable instances separately for finding swaps
	var druInstances, josieInstances []InstanceWithTask
	currentDruTotal := 0.0
	currentJosieTotal := 0.0

	for _, inst := range instances {
		if inst.Instance.WeekStartDate.Equal(week) {
			if inst.Instance.AssignedTo == "dru" {
				currentDruTotal += float64(inst.EstimatedMins)
				if inst.CanSwap {
					druInstances = append(druInstances, inst)
				}
			} else if inst.Instance.AssignedTo == "josie" {
				currentJosieTotal += float64(inst.EstimatedMins)
				if inst.CanSwap {
					josieInstances = append(josieInstances, inst)
				}
			}
		}
	}

	if len(druInstances) == 0 || len(josieInstances) == 0 {
		// Can't swap if one person has no swappable tasks this week
		return false, nil
	}

	currentDiff := math.Abs(currentDruTotal - currentJosieTotal)

	// Find the best swap (including no swap)
	bestDiff := currentDiff
	var bestDruIdx, bestJosieIdx int = -1, -1

	// Try all possible swaps
	for di, dInst := range druInstances {
		for ji, jInst := range josieInstances {
			// Calculate new totals after swap
			newDruTotal := currentDruTotal - float64(dInst.EstimatedMins) + float64(jInst.EstimatedMins)
			newJosieTotal := currentJosieTotal - float64(jInst.EstimatedMins) + float64(dInst.EstimatedMins)
			newDiff := math.Abs(newDruTotal - newJosieTotal)

			// Is this swap better?
			if newDiff < bestDiff {
				bestDiff = newDiff
				bestDruIdx = di
				bestJosieIdx = ji
			}
		}
	}

	// If we found a beneficial swap, perform it
	if bestDruIdx >= 0 && bestJosieIdx >= 0 {
		druInst := druInstances[bestDruIdx]
		josieInst := josieInstances[bestJosieIdx]

		// Swap assignments in database
		err := db.UpdateTaskInstanceAssignment(ctx, sqlc.UpdateTaskInstanceAssignmentParams{
			ID:         druInst.Instance.ID,
			AssignedTo: "josie",
		})
		if err != nil {
			return false, fmt.Errorf("failed to update dru instance: %w", err)
		}

		err = db.UpdateTaskInstanceAssignment(ctx, sqlc.UpdateTaskInstanceAssignmentParams{
			ID:         josieInst.Instance.ID,
			AssignedTo: "dru",
		})
		if err != nil {
			return false, fmt.Errorf("failed to update josie instance: %w", err)
		}

		// Update in-memory instances for next iteration
		for i := range instances {
			if instances[i].Instance.ID == druInst.Instance.ID {
				instances[i].Instance.AssignedTo = "josie"
			}
			if instances[i].Instance.ID == josieInst.Instance.ID {
				instances[i].Instance.AssignedTo = "dru"
			}
		}

		return true, nil
	}

	return false, nil
}
