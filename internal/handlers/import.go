package handlers

import (
	"database/sql"
	"encoding/csv"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/druarnfield/cleaning-scheduler/internal/scheduler"
	templPages "github.com/druarnfield/cleaning-scheduler/internal/templates/pages"
)

func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		user := auth.GetUserFromContext(r.Context())
		component := templPages.ImportPage(user)
		Render(w, r, component)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("csvfile")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)

	// Skip header
	_, err = reader.Read()
	if err != nil {
		http.Error(w, "Invalid CSV", http.StatusBadRequest)
		return
	}

	var tasksCreated int
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(record) < 5 {
			continue
		}

		name := strings.TrimSpace(record[0])
		category := strings.TrimSpace(record[1])
		frequency := strings.TrimSpace(record[2])
		assignedTo := strings.TrimSpace(record[3])
		estimatedMins, _ := strconv.Atoi(strings.TrimSpace(record[4]))

		if name == "" || category == "" {
			continue
		}

		// Determine default assignee
		var defaultAssignee sql.NullString
		if assignedTo != "" {
			defaultAssignee = sql.NullString{String: assignedTo, Valid: true}
		}

		// Create task
		_, err = h.db.CreateTask(r.Context(), sqlc.CreateTaskParams{
			Name:            name,
			Category:        category,
			Frequency:       frequency,
			EstimatedMins:   int64(estimatedMins),
			DefaultAssignee: defaultAssignee,
		})

		if err == nil {
			tasksCreated++
		}
	}

	// Generate initial instances
	startDate := time.Now().Truncate(24 * time.Hour)
	scheduler.GenerateInstances(r.Context(), h.db, startDate, 4)

	// Redirect to schedule
	http.Redirect(w, r, "/schedule", http.StatusSeeOther)
}