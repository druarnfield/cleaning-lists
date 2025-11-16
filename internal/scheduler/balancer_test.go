package scheduler

import (
	"database/sql"
	"testing"
	"time"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
)

func TestCalculateWeeklyWorkload(t *testing.T) {
	// Setup test data
	week1 := time.Date(2025, 11, 17, 0, 0, 0, 0, time.UTC)
	week2 := time.Date(2025, 11, 24, 0, 0, 0, 0, time.UTC)

	instances := []InstanceWithTask{
		{
			Instance: sqlc.TaskInstance{
				ID:            1,
				WeekStartDate: week1,
				AssignedTo:    "dru",
			},
			EstimatedMins: 30,
		},
		{
			Instance: sqlc.TaskInstance{
				ID:            2,
				WeekStartDate: week1,
				AssignedTo:    "josie",
			},
			EstimatedMins: 60,
		},
		{
			Instance: sqlc.TaskInstance{
				ID:            3,
				WeekStartDate: week2,
				AssignedTo:    "dru",
			},
			EstimatedMins: 90,
		},
	}

	workloads := calculateWeeklyWorkload(instances)

	// Verify week 1
	if wl, exists := workloads[week1]; !exists {
		t.Errorf("Expected workload for week 1, got none")
	} else {
		if wl.DruMins != 30 {
			t.Errorf("Expected Dru to have 30 mins in week 1, got %.2f", wl.DruMins)
		}
		if wl.JosieMins != 60 {
			t.Errorf("Expected Josie to have 60 mins in week 1, got %.2f", wl.JosieMins)
		}
		if wl.Difference != 30 {
			t.Errorf("Expected difference of 30 mins in week 1, got %.2f", wl.Difference)
		}
	}

	// Verify week 2
	if wl, exists := workloads[week2]; !exists {
		t.Errorf("Expected workload for week 2, got none")
	} else {
		if wl.DruMins != 90 {
			t.Errorf("Expected Dru to have 90 mins in week 2, got %.2f", wl.DruMins)
		}
		if wl.JosieMins != 0 {
			t.Errorf("Expected Josie to have 0 mins in week 2, got %.2f", wl.JosieMins)
		}
		if wl.Difference != 90 {
			t.Errorf("Expected difference of 90 mins in week 2, got %.2f", wl.Difference)
		}
	}
}

func TestFindMostImbalancedWeek(t *testing.T) {
	week1 := time.Date(2025, 11, 17, 0, 0, 0, 0, time.UTC)
	week2 := time.Date(2025, 11, 24, 0, 0, 0, 0, time.UTC)
	week3 := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)

	workloads := map[time.Time]*WeekWorkload{
		week1: {
			Week:       week1,
			DruMins:    100,
			JosieMins:  80,
			Difference: 20,
		},
		week2: {
			Week:       week2,
			DruMins:    200,
			JosieMins:  100,
			Difference: 100, // Most imbalanced
		},
		week3: {
			Week:       week3,
			DruMins:    150,
			JosieMins:  120,
			Difference: 30,
		},
	}

	mostImbalanced := findMostImbalancedWeek(workloads)

	if mostImbalanced == nil {
		t.Fatalf("Expected most imbalanced week, got nil")
	}

	if !mostImbalanced.Week.Equal(week2) {
		t.Errorf("Expected week2 to be most imbalanced, got %v", mostImbalanced.Week)
	}

	if mostImbalanced.Difference != 100 {
		t.Errorf("Expected difference of 100, got %.2f", mostImbalanced.Difference)
	}
}

func TestCanSwapLogic(t *testing.T) {
	tests := []struct {
		name            string
		defaultAssignee sql.NullString
		expectedCanSwap bool
	}{
		{
			name:            "Alternate tasks can swap",
			defaultAssignee: sql.NullString{String: "alternate", Valid: true},
			expectedCanSwap: true,
		},
		{
			name:            "NULL default assignee can swap",
			defaultAssignee: sql.NullString{Valid: false},
			expectedCanSwap: true,
		},
		{
			name:            "Empty string can swap",
			defaultAssignee: sql.NullString{String: "", Valid: true},
			expectedCanSwap: true,
		},
		{
			name:            "Pre-assigned to Dru cannot swap",
			defaultAssignee: sql.NullString{String: "dru", Valid: true},
			expectedCanSwap: false,
		},
		{
			name:            "Pre-assigned to Josie cannot swap",
			defaultAssignee: sql.NullString{String: "josie", Valid: true},
			expectedCanSwap: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := sqlc.Task{
				DefaultAssignee: tt.defaultAssignee,
			}

			canSwap := !task.DefaultAssignee.Valid ||
				task.DefaultAssignee.String == "" ||
				task.DefaultAssignee.String == "alternate"

			if canSwap != tt.expectedCanSwap {
				t.Errorf("Expected canSwap=%v, got %v", tt.expectedCanSwap, canSwap)
			}
		})
	}
}

func TestSwapCalculation(t *testing.T) {
	// Test the math of swapping tasks to reduce imbalance
	tests := []struct {
		name          string
		druCurrent    float64
		josieCurrent  float64
		druTaskMins   float64
		josieTaskMins float64
		shouldSwap    bool
	}{
		{
			name:          "Swap reduces imbalance",
			druCurrent:    200, // Dru has more work
			josieCurrent:  100,
			druTaskMins:   60, // Swap 60-min task from Dru
			josieTaskMins: 30, // With 30-min task from Josie
			shouldSwap:    true,
			// After swap: Dru=170, Josie=130, diff=40 (better than 100)
		},
		{
			name:          "Swap makes imbalance worse",
			druCurrent:    150,
			josieCurrent:  140,
			druTaskMins:   10, // Swap 10-min from Dru
			josieTaskMins: 50, // With 50-min from Josie
			shouldSwap:    false,
			// After swap: Dru=190, Josie=100, diff=90 (worse than 10)
		},
		{
			name:          "Equal swap with imbalance",
			druCurrent:    200,
			josieCurrent:  100,
			druTaskMins:   30,
			josieTaskMins: 30,
			shouldSwap:    false,
			// After swap: Dru=200, Josie=100, diff=100 (same, no benefit)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentDiff := abs(tt.druCurrent - tt.josieCurrent)

			// Calculate difference after swap
			newDruTotal := tt.druCurrent - tt.druTaskMins + tt.josieTaskMins
			newJosieTotal := tt.josieCurrent - tt.josieTaskMins + tt.druTaskMins
			newDiff := abs(newDruTotal - newJosieTotal)

			shouldSwap := newDiff < currentDiff

			if shouldSwap != tt.shouldSwap {
				t.Errorf("Expected shouldSwap=%v, got %v (current diff=%.0f, new diff=%.0f)",
					tt.shouldSwap, shouldSwap, currentDiff, newDiff)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestBalanceThreshold(t *testing.T) {
	// Test that the balance threshold constant is set correctly
	if BalanceThreshold != 30.0 {
		t.Errorf("Expected BalanceThreshold to be 30.0, got %.2f", BalanceThreshold)
	}
}

func TestMaxSwapIterations(t *testing.T) {
	// Test that max iterations is set to prevent infinite loops
	if MaxSwapIterations <= 0 {
		t.Errorf("MaxSwapIterations must be positive, got %d", MaxSwapIterations)
	}
	if MaxSwapIterations > 1000 {
		t.Errorf("MaxSwapIterations seems too high (%d), may cause performance issues", MaxSwapIterations)
	}
}

// TestWeekWorkloadStructure tests the WeekWorkload structure
func TestWeekWorkloadStructure(t *testing.T) {
	week := time.Date(2025, 11, 17, 0, 0, 0, 0, time.UTC)
	wl := &WeekWorkload{
		Week:       week,
		DruMins:    150,
		JosieMins:  100,
		Difference: 50,
		Percent:    20.0,
	}

	if !wl.Week.Equal(week) {
		t.Errorf("Week mismatch")
	}
	if wl.DruMins != 150 {
		t.Errorf("DruMins mismatch")
	}
	if wl.JosieMins != 100 {
		t.Errorf("JosieMins mismatch")
	}
	if wl.Difference != 50 {
		t.Errorf("Difference mismatch")
	}
}

// TestInstanceWithTaskStructure tests the InstanceWithTask structure
func TestInstanceWithTaskStructure(t *testing.T) {
	inst := InstanceWithTask{
		Instance: sqlc.TaskInstance{
			ID:         1,
			AssignedTo: "dru",
		},
		Task: sqlc.Task{
			ID:   10,
			Name: "Test Task",
		},
		EstimatedMins: 30,
		CanSwap:       true,
	}

	if inst.Instance.ID != 1 {
		t.Errorf("Instance ID mismatch")
	}
	if inst.Task.ID != 10 {
		t.Errorf("Task ID mismatch")
	}
	if inst.EstimatedMins != 30 {
		t.Errorf("EstimatedMins mismatch")
	}
	if !inst.CanSwap {
		t.Errorf("CanSwap should be true")
	}
}
