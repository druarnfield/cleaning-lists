package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"
	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/druarnfield/cleaning-scheduler/internal/llm"
	"github.com/druarnfield/cleaning-scheduler/internal/scheduler"
	templPages "github.com/druarnfield/cleaning-scheduler/internal/templates/pages"
	"github.com/go-chi/chi/v5"
)

// Local types for handler logic - map to templ types
type localWeekData struct {
	WeekLabel      string
	StartDate      string
	EndDate        string
	DruTasks       []localTaskDisplay
	JosieTasks     []localTaskDisplay
	DruTaskCount   int
	DruMins        int
	JosieTaskCount int
	JosieMins      int
}

type localTaskDisplay struct {
	ID             int64
	TaskID         int64
	TaskName       string
	Category       string
	EstimatedMins  int
	AssignedTo     string
	IsCompleted    bool
	BroughtForward bool
	OriginalDate   string
	IsPast         bool
}

func (h *Handler) ScheduleView(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// Get week offset from query params
	weekOffset := 0
	if weekStr := r.URL.Query().Get("week"); weekStr != "" {
		if offset, err := strconv.Atoi(weekStr); err == nil {
			weekOffset = offset
		}
	}

	// Get assignee filter from query params
	assigneeFilter := r.URL.Query().Get("assignee")
	if assigneeFilter == "" {
		assigneeFilter = "all"
	}

	// Calculate the target week based on offset
	now := time.Now()
	currentWeekStart := scheduler.GetWeekStart(now)
	targetWeekStart := currentWeekStart.AddDate(0, 0, weekOffset*7)

	// Get instances for target week
	weekInstances, _ := h.db.ListInstancesByWeek(r.Context(), targetWeekStart)

	// Convert to templ data structure with filter
	weekData := h.formatWeekDataForTemplWithFilter(r.Context(), weekInstances, targetWeekStart, assigneeFilter)
	weekData.WeekOffset = weekOffset
	weekData.IsCurrentWeek = (weekOffset == 0)

	data := templPages.SchedulePageData{
		CurrentWeek: weekData,
		WeekOffset:  weekOffset,
	}

	// If this is an HTMX request, return the schedule content (minutes + week)
	if r.Header.Get("HX-Request") == "true" {
		component := templPages.ScheduleContent(data)
		Render(w, r, component)
		return
	}

	// Otherwise return the full page
	component := templPages.SchedulePage(user, data)
	Render(w, r, component)
}

func (h *Handler) formatWeekData(ctx context.Context, instances []sqlc.ListInstancesByWeekRow, weekStart time.Time) localWeekData {
	weekEnd := weekStart.AddDate(0, 0, 6)
	now := time.Now()
	currentWeekStart := scheduler.GetWeekStart(now)

	weekLabel := "Week"
	if weekStart.Equal(currentWeekStart) {
		weekLabel = "Current Week"
	} else if weekStart.Before(currentWeekStart) {
		weekLabel = "Previous Week"
	} else {
		weekLabel = "Next Week"
	}

	// Separate by assignee
	druTasks := []localTaskDisplay{}
	josieTasks := []localTaskDisplay{}
	druMins := 0
	josieMins := 0

	// Check completions from the query results
	for _, inst := range instances {
		isCompleted := inst.CompletionID.Valid

		taskDisplay := localTaskDisplay{
			ID:             inst.ID,
			TaskID:         inst.TaskID,
			TaskName:       inst.TaskName,
			Category:       inst.TaskCategory,
			EstimatedMins:  int(inst.EstimatedMins),
			AssignedTo:     inst.AssignedTo,
			IsCompleted:    isCompleted,
			BroughtForward: inst.BroughtForward.Bool,
			IsPast:         inst.ScheduledDate.Before(time.Now().Truncate(24 * time.Hour)),
		}

		if inst.OriginalScheduledDate.Valid {
			taskDisplay.OriginalDate = inst.OriginalScheduledDate.Time.Format("Jan 2")
		}

		if inst.AssignedTo == "dru" {
			druTasks = append(druTasks, taskDisplay)
			if !isCompleted && inst.CountsTowardWeekly.Bool {
				druMins += int(inst.EstimatedMins)
			}
		} else {
			josieTasks = append(josieTasks, taskDisplay)
			if !isCompleted && inst.CountsTowardWeekly.Bool {
				josieMins += int(inst.EstimatedMins)
			}
		}
	}

	return localWeekData{
		WeekLabel:      weekLabel,
		StartDate:      weekStart.Format("Jan 2"),
		EndDate:        weekEnd.Format("Jan 2"),
		DruTasks:       druTasks,
		JosieTasks:     josieTasks,
		DruTaskCount:   len(druTasks),
		DruMins:        druMins,
		JosieTaskCount: len(josieTasks),
		JosieMins:      josieMins,
	}
}

// Convert local week data to templ structure
func (h *Handler) formatWeekDataForTempl(ctx context.Context, instances []sqlc.ListInstancesByWeekRow, weekStart time.Time) templPages.WeekData {
	return h.formatWeekDataForTemplWithFilter(ctx, instances, weekStart, "all")
}

// Convert local week data to templ structure with assignee filter
func (h *Handler) formatWeekDataForTemplWithFilter(ctx context.Context, instances []sqlc.ListInstancesByWeekRow, weekStart time.Time, assigneeFilter string) templPages.WeekData {
	tasks := []templPages.TaskDisplay{}
	druRemaining := 0
	josieRemaining := 0

	for _, inst := range instances {
		isCompleted := inst.CompletionID.Valid

		// Apply assignee filter
		if assigneeFilter != "all" && inst.AssignedTo != assigneeFilter {
			// Still count towards remaining minutes but don't display
			if !isCompleted {
				if inst.AssignedTo == "dru" {
					druRemaining += int(inst.EstimatedMins)
				} else if inst.AssignedTo == "josie" {
					josieRemaining += int(inst.EstimatedMins)
				}
			}
			continue
		}

		task := templPages.TaskDisplay{
			ID:             inst.ID,
			Name:           inst.TaskName,
			Category:       inst.TaskCategory,
			AssignedTo:     inst.AssignedTo,
			EstimatedMins:  inst.EstimatedMins,
			ScheduledDate:  inst.ScheduledDate.Format("Jan 2"),
			IsCompleted:    isCompleted,
			CanBringForward: !inst.BroughtForward.Bool && !isCompleted,
		}
		tasks = append(tasks, task)

		// Calculate remaining minutes for incomplete tasks
		if !isCompleted {
			if inst.AssignedTo == "dru" {
				druRemaining += int(inst.EstimatedMins)
			} else if inst.AssignedTo == "josie" {
				josieRemaining += int(inst.EstimatedMins)
			}
		}
	}

	return templPages.WeekData{
		WeekStart: weekStart.Format("Jan 2, 2006"),
		Tasks:     tasks,
		DruRemainingMins: druRemaining,
		JosieRemainingMins: josieRemaining,
	}
}

func (h *Handler) ToggleCompletion(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	instanceID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid instance ID", http.StatusBadRequest)
		return
	}

	// Check if completion exists
	_, err = h.db.GetCompletionByInstance(r.Context(), instanceID)

	if err == sql.ErrNoRows {
		// Create completion
		_, err = h.db.CreateCompletion(r.Context(), sqlc.CreateCompletionParams{
			TaskInstanceID: instanceID,
			CompletedBy:    user.Username,
		})
		if err != nil {
			http.Error(w, "Failed to create completion", http.StatusInternalServerError)
			return
		}
	} else if err == nil {
		// Delete completion
		err = h.db.DeleteCompletion(r.Context(), instanceID)
		if err != nil {
			http.Error(w, "Failed to delete completion", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Failed to check completion status", http.StatusInternalServerError)
		return
	}

	// Get the instance to determine the week
	inst, err := h.db.GetInstanceWithTask(r.Context(), instanceID)
	if err != nil {
		http.Error(w, "Failed to get updated task", http.StatusInternalServerError)
		return
	}

	// Get the week offset
	now := time.Now()
	currentWeekStart := scheduler.GetWeekStart(now)
	instanceWeekStart := scheduler.GetWeekStart(inst.ScheduledDate)
	weekOffset := int(instanceWeekStart.Sub(currentWeekStart).Hours() / 24 / 7)

	// Get all instances for the week to recalculate totals
	weekInstances, _ := h.db.ListInstancesByWeek(r.Context(), instanceWeekStart)

	// Convert to templ data structure
	weekData := h.formatWeekDataForTempl(r.Context(), weekInstances, instanceWeekStart)
	weekData.WeekOffset = weekOffset
	weekData.IsCurrentWeek = (weekOffset == 0)

	data := templPages.SchedulePageData{
		CurrentWeek: weekData,
		WeekOffset:  weekOffset,
	}

	// Return the entire schedule content to update minutes
	component := templPages.ScheduleContent(data)
	Render(w, r, component)
}

func (h *Handler) BringForwardTask(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	taskID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	targetDate := r.FormValue("target_date")
	if targetDate == "" {
		http.Error(w, "Target date required", http.StatusBadRequest)
		return
	}

	// Parse target date
	target, err := time.Parse("2006-01-02", targetDate)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	// Bring forward the task
	err = scheduler.BringForwardTask(r.Context(), h.db, taskID, target, user.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to schedule
	http.Redirect(w, r, "/schedule", http.StatusSeeOther)
}

func (h *Handler) GenerateSchedule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("Manual LLM-powered task generation triggered...")
	weeksAhead := llm.ParseWeeksAhead()
	err := llm.GenerateSchedule(r.Context(), h.db, time.Now(), weeksAhead)
	if err != nil {
		log.Printf("Error generating instances: %v", err)
		http.Error(w, "Failed to generate schedule", http.StatusInternalServerError)
		return
	}

	// If HTMX request, return the schedule content
	if r.Header.Get("HX-Request") == "true" {
		// Get current week data
		now := time.Now()
		currentWeekStart := scheduler.GetWeekStart(now)
		weekInstances, _ := h.db.ListInstancesByWeek(r.Context(), currentWeekStart)

		weekData := h.formatWeekDataForTempl(r.Context(), weekInstances, currentWeekStart)
		weekData.WeekOffset = 0
		weekData.IsCurrentWeek = true

		data := templPages.SchedulePageData{
			CurrentWeek: weekData,
			WeekOffset:  0,
		}

		component := templPages.ScheduleContent(data)
		Render(w, r, component)
		return
	}

	// Otherwise redirect back to schedule
	http.Redirect(w, r, "/schedule", http.StatusSeeOther)
}