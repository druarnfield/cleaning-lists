package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"
	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/druarnfield/cleaning-scheduler/internal/llm"
	templPages "github.com/druarnfield/cleaning-scheduler/internal/templates/pages"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) TasksList(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	tasks, err := h.db.GetTasksWithNextScheduled(r.Context())
	if err != nil {
		http.Error(w, "Failed to list tasks", http.StatusInternalServerError)
		return
	}

	// Convert sqlc tasks to templ task items
	taskItems := make([]templPages.TaskItem, len(tasks))
	for i, task := range tasks {
		taskItem := templPages.TaskItem{
			ID:              task.ID,
			Name:            task.Name,
			Category:        task.Category,
			Frequency:       task.Frequency,
			DefaultAssignee: task.DefaultAssignee,
			EstimatedMins:   task.EstimatedMins,
		}

		// Handle nullable NextScheduledDate
		// SQLite returns dates as strings, so we need to parse them
		if task.NextScheduledDate != nil {
			if dateStr, ok := task.NextScheduledDate.(string); ok && dateStr != "" {
				// Parse SQLite datetime format (YYYY-MM-DD HH:MM:SS+TZ:TZ)
				if parsedTime, err := time.Parse("2006-01-02 15:04:05-07:00", dateStr); err == nil {
					taskItem.NextScheduledDate = sql.NullTime{Time: parsedTime, Valid: true}
				}
			}
		}

		// Handle nullable NextAssignedTo
		if task.NextAssignedTo != nil {
			if s, ok := task.NextAssignedTo.(string); ok && s != "" {
				taskItem.NextAssignedTo = sql.NullString{String: s, Valid: true}
			}
		}

		taskItems[i] = taskItem
	}

	data := templPages.TasksPageData{
		Tasks: taskItems,
	}

	component := templPages.TasksPage(user, data)
	Render(w, r, component)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	category := r.FormValue("category")
	frequency := r.FormValue("frequency")
	assignedTo := r.FormValue("assigned_to")
	estimatedMinsStr := r.FormValue("estimated_mins")

	estimatedMins, _ := strconv.ParseInt(estimatedMinsStr, 10, 64)

	var defaultAssignee sql.NullString
	if assignedTo != "" {
		defaultAssignee = sql.NullString{String: assignedTo, Valid: true}
	}

	task, err := h.db.CreateTask(r.Context(), sqlc.CreateTaskParams{
		Name:            name,
		Category:        category,
		Frequency:       frequency,
		EstimatedMins:   estimatedMins,
		DefaultAssignee: defaultAssignee,
	})

	if err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	// Regenerate schedule with LLM to include the new task
	weeksAhead := llm.ParseWeeksAhead()
	err = llm.GenerateSchedule(r.Context(), h.db, time.Now(), weeksAhead)
	if err != nil {
		log.Printf("Warning: Failed to regenerate schedule for new task %d: %v", task.ID, err)
		// Don't fail the request - task is created, instances can be generated later
	}

	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	category := r.FormValue("category")
	frequency := r.FormValue("frequency")
	assignedTo := r.FormValue("assigned_to")
	estimatedMinsStr := r.FormValue("estimated_mins")

	estimatedMins, _ := strconv.ParseInt(estimatedMinsStr, 10, 64)

	var defaultAssignee sql.NullString
	if assignedTo != "" {
		defaultAssignee = sql.NullString{String: assignedTo, Valid: true}
	}

	_, err = h.db.UpdateTask(r.Context(), sqlc.UpdateTaskParams{
		ID:              taskID,
		Name:            name,
		Category:        category,
		Frequency:       frequency,
		EstimatedMins:   estimatedMins,
		DefaultAssignee: defaultAssignee,
	})

	if err != nil {
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") != "" {
		// HTMX request - return updated task row
		task, _ := h.db.GetTask(r.Context(), taskID)
		taskItem := templPages.TaskItem{
			ID:              task.ID,
			Name:            task.Name,
			Category:        task.Category,
			Frequency:       task.Frequency,
			DefaultAssignee: task.DefaultAssignee,
			EstimatedMins:   task.EstimatedMins,
		}
		component := templPages.TaskRowEdit(taskItem)
		Render(w, r, component)
	} else {
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
	}
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	err = h.db.DeleteTask(r.Context(), taskID)
	if err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") != "" {
		// HTMX request - return empty response to remove element
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
	}
}