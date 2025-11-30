package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/druarnfield/cleaning-scheduler/internal/llm"
)

type ChatRequest struct {
	Message    string `json:"message"`
	WeekOffset int    `json:"week_offset"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Updated  bool   `json:"updated"`
}

func (h *Handler) ScheduleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	log.Printf("Received chat request for week %d: %s", req.WeekOffset, req.Message)

	// Call LLM to adjust schedule
	response, updated, err := llm.AdjustSchedule(r.Context(), h.db, req.Message, req.WeekOffset)
	if err != nil {
		log.Printf("Error adjusting schedule: %v", err)
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	// Save to chat history
	_, err = h.db.CreateScheduleSuggestion(r.Context(), sqlc.CreateScheduleSuggestionParams{
		UserMessage: req.Message,
		LlmResponse: response,
		WeekOffset: sql.NullInt64{
			Int64: int64(req.WeekOffset),
			Valid: true,
		},
		ChangesMade: sql.NullBool{
			Bool:  updated,
			Valid: true,
		},
	})
	if err != nil {
		log.Printf("Warning: Failed to save chat history: %v", err)
		// Don't fail the request
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{
		Response: response,
		Updated:  updated,
	})
}

func (h *Handler) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := int64(10) // Default limit

	if limitStr != "" {
		parsedLimit, err := strconv.ParseInt(limitStr, 10, 64)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	suggestions, err := h.db.GetRecentSuggestions(r.Context(), limit)
	if err != nil {
		log.Printf("Error getting chat history: %v", err)
		http.Error(w, "Failed to get chat history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}
