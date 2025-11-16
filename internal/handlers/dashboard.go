package handlers

import (
	"net/http"
	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/druarnfield/cleaning-scheduler/internal/scheduler"
	templPages "github.com/druarnfield/cleaning-scheduler/internal/templates/pages"
)

func (h *Handler) DashboardView(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// Get workload summary
	druMins, josieMins, err := scheduler.GetWorkloadSummary(r.Context(), h.db)
	if err != nil {
		http.Error(w, "Failed to get workload summary", http.StatusInternalServerError)
		return
	}

	// Get completion stats for current week
	completionStats, err := h.db.GetWeeklyCompletionStats(r.Context())
	if err != nil {
		completionStats = []sqlc.GetWeeklyCompletionStatsRow{}
	}

	// Get task counts
	totalTasks, _ := h.db.CountTasks(r.Context())
	totalInstances, _ := h.db.CountInstances(r.Context())

	// Convert completion stats to templ format
	completionStatsForTempl := make([]templPages.CompletionStat, len(completionStats))
	for i, stat := range completionStats {
		completedCount := int64(0)
		if stat.CompletedCount.Valid {
			completedCount = int64(stat.CompletedCount.Float64)
		}
		completionStatsForTempl[i] = templPages.CompletionStat{
			WeekStart:      stat.WeekStart.Format("Jan 2"),
			CompletedCount: completedCount,
			TotalCount:     stat.TotalCount,
		}
	}

	data := templPages.DashboardData{
		DruMins:         int64(druMins),
		DruHours:        druMins / 60.0,
		JosieMins:       int64(josieMins),
		JosieHours:      josieMins / 60.0,
		TotalTasks:      totalTasks,
		TotalInstances:  totalInstances,
		CompletionStats: completionStatsForTempl,
	}

	component := templPages.DashboardPage(user, data)
	Render(w, r, component)
}