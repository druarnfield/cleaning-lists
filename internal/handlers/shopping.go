package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/druarnfield/cleaning-scheduler/internal/scheduler"
	templPages "github.com/druarnfield/cleaning-scheduler/internal/templates/pages"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) ShoppingListView(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// Parse week offset from query parameter
	weekOffset := 0
	if weekStr := r.URL.Query().Get("week"); weekStr != "" {
		if offset, err := strconv.Atoi(weekStr); err == nil {
			weekOffset = offset
		}
	}

	// Calculate target week start date
	now := time.Now()
	currentWeekStart := scheduler.GetWeekStart(now)
	targetWeekStart := currentWeekStart.AddDate(0, 0, weekOffset*7)

	// Fetch shopping items for the target week
	items, err := h.db.ListShoppingItemsByWeek(r.Context(), targetWeekStart)
	if err != nil {
		log.Printf("Error listing shopping items: %v", err)
		items = []sqlc.ShoppingItem{}
	}

	// Group items by meal
	itemsByMeal := make(map[string][]sqlc.ShoppingItem)
	var noMealItems []sqlc.ShoppingItem

	for _, item := range items {
		if item.Meal.Valid && item.Meal.String != "" {
			meal := item.Meal.String
			itemsByMeal[meal] = append(itemsByMeal[meal], item)
		} else {
			noMealItems = append(noMealItems, item)
		}
	}

	// Determine week label
	weekLabel := "Current Week"
	if weekOffset < 0 {
		weekLabel = "Previous Week"
	} else if weekOffset > 0 {
		weekLabel = "Next Week"
	}

	data := templPages.ShoppingPageData{
		WeekStart:     targetWeekStart.Format("2006-01-02"),
		WeekLabel:     weekLabel,
		WeekOffset:    weekOffset,
		ItemsByMeal:   itemsByMeal,
		NoMealItems:   noMealItems,
		IsCurrentWeek: targetWeekStart.Equal(currentWeekStart),
	}

	// Return HTMX partial or full page
	if r.Header.Get("HX-Request") == "true" {
		component := templPages.ShoppingContent(data)
		Render(w, r, component)
		return
	}

	component := templPages.ShoppingPage(user, data)
	Render(w, r, component)
}

func (h *Handler) CreateShoppingItem(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	name := strings.TrimSpace(r.FormValue("name"))
	meal := strings.TrimSpace(r.FormValue("meal"))
	quantityStr := r.FormValue("quantity")
	weekStartStr := r.FormValue("week_start")

	if name == "" {
		http.Error(w, "Item name is required", http.StatusBadRequest)
		return
	}

	quantity := int64(1)
	if quantityStr != "" {
		if q, err := strconv.ParseInt(quantityStr, 10, 64); err == nil && q > 0 {
			quantity = q
		}
	}

	// Parse week start date
	weekStart, err := time.Parse("2006-01-02", weekStartStr)
	if err != nil {
		http.Error(w, "Invalid week start date", http.StatusBadRequest)
		return
	}

	// Create the shopping item
	var mealNull sql.NullString
	if meal != "" {
		mealNull = sql.NullString{String: meal, Valid: true}

		// Auto-create meal if it doesn't exist
		_, err = h.db.GetMealByName(r.Context(), meal)
		if err == sql.ErrNoRows {
			_, err = h.db.CreateMeal(r.Context(), meal)
			if err != nil {
				log.Printf("Warning: Failed to auto-create meal: %v", err)
			}
		}
	}

	_, err = h.db.CreateShoppingItem(r.Context(), sqlc.CreateShoppingItemParams{
		Name:          name,
		Meal:          mealNull,
		Quantity:      quantity,
		WeekStartDate: weekStart,
		AddedBy:       user.Username,
	})

	if err != nil {
		log.Printf("Error creating shopping item: %v", err)
		http.Error(w, "Failed to create item", http.StatusInternalServerError)
		return
	}

	// Update item history for autocomplete
	err = h.db.UpsertItemHistory(r.Context(), sqlc.UpsertItemHistoryParams{
		Name: name,
		Meal: mealNull,
	})
	if err != nil {
		log.Printf("Warning: Failed to update item history: %v", err)
	}

	// Redirect back to shopping list with current week offset
	weekOffset := 0
	if weekOffsetStr := r.FormValue("week_offset"); weekOffsetStr != "" {
		if offset, err := strconv.Atoi(weekOffsetStr); err == nil {
			weekOffset = offset
		}
	}

	http.Redirect(w, r, "/shopping?week="+strconv.Itoa(weekOffset), http.StatusSeeOther)
}

func (h *Handler) QuickAddMeal(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	meal := chi.URLParam(r, "meal")
	weekStartStr := r.FormValue("week_start")

	if meal == "" {
		http.Error(w, "Meal name is required", http.StatusBadRequest)
		return
	}

	// Parse week start date
	weekStart, err := time.Parse("2006-01-02", weekStartStr)
	if err != nil {
		http.Error(w, "Invalid week start date", http.StatusBadRequest)
		return
	}

	// Get items for this meal
	items, err := h.db.GetItemsByMeal(r.Context(), meal)
	if err != nil {
		log.Printf("Error getting items by meal: %v", err)
		http.Error(w, "Failed to get meal items", http.StatusInternalServerError)
		return
	}

	// Add each item to the shopping list
	for _, item := range items {
		_, err = h.db.CreateShoppingItem(r.Context(), sqlc.CreateShoppingItemParams{
			Name:          item.Name,
			Meal:          item.Meal,
			Quantity:      item.Quantity,
			WeekStartDate: weekStart,
			AddedBy:       user.Username,
		})
		if err != nil {
			log.Printf("Warning: Failed to add item %s: %v", item.Name, err)
		}

		// Update history
		h.db.UpsertItemHistory(r.Context(), sqlc.UpsertItemHistoryParams{
			Name: item.Name,
			Meal: item.Meal,
		})
	}

	// Redirect back to shopping list
	weekOffset := 0
	if weekOffsetStr := r.FormValue("week_offset"); weekOffsetStr != "" {
		if offset, err := strconv.Atoi(weekOffsetStr); err == nil {
			weekOffset = offset
		}
	}

	http.Redirect(w, r, "/shopping?week="+strconv.Itoa(weekOffset), http.StatusSeeOther)
}

func (h *Handler) UpdateShoppingItem(w http.ResponseWriter, r *http.Request) {
	itemIDStr := chi.URLParam(r, "id")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	meal := strings.TrimSpace(r.FormValue("meal"))
	quantityStr := r.FormValue("quantity")

	if name == "" {
		http.Error(w, "Item name is required", http.StatusBadRequest)
		return
	}

	quantity := int64(1)
	if quantityStr != "" {
		if q, err := strconv.ParseInt(quantityStr, 10, 64); err == nil && q > 0 {
			quantity = q
		}
	}

	var mealNull sql.NullString
	if meal != "" {
		mealNull = sql.NullString{String: meal, Valid: true}
	}

	_, err = h.db.UpdateShoppingItem(r.Context(), sqlc.UpdateShoppingItemParams{
		ID:       itemID,
		Name:     name,
		Meal:     mealNull,
		Quantity: quantity,
	})

	if err != nil {
		log.Printf("Error updating shopping item: %v", err)
		http.Error(w, "Failed to update item", http.StatusInternalServerError)
		return
	}

	// Update history
	h.db.UpsertItemHistory(r.Context(), sqlc.UpsertItemHistoryParams{
		Name: name,
		Meal: mealNull,
	})

	// Return updated item row for HTMX
	item, _ := h.db.GetShoppingItem(r.Context(), itemID)

	// Calculate week offset from current week
	now := time.Now()
	currentWeekStart := scheduler.GetWeekStart(now)
	itemWeekStart := scheduler.GetWeekStart(item.WeekStartDate)
	weekOffset := int(itemWeekStart.Sub(currentWeekStart).Hours() / 24 / 7)

	component := templPages.ShoppingItemRow(item, item.WeekStartDate.Format("2006-01-02"), weekOffset)
	Render(w, r, component)
}

func (h *Handler) DeleteShoppingItem(w http.ResponseWriter, r *http.Request) {
	itemIDStr := chi.URLParam(r, "id")
	itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid item ID", http.StatusBadRequest)
		return
	}

	err = h.db.DeleteShoppingItem(r.Context(), itemID)
	if err != nil {
		log.Printf("Error deleting shopping item: %v", err)
		http.Error(w, "Failed to delete item", http.StatusInternalServerError)
		return
	}

	// Return empty response to remove element (HTMX handles this)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteShoppingItemsByMeal(w http.ResponseWriter, r *http.Request) {
	meal := chi.URLParam(r, "meal")
	weekStartStr := r.URL.Query().Get("week_start")

	if meal == "" {
		http.Error(w, "Meal name is required", http.StatusBadRequest)
		return
	}

	// Parse week start date
	weekStart, err := time.Parse("2006-01-02", weekStartStr)
	if err != nil {
		http.Error(w, "Invalid week start date", http.StatusBadRequest)
		return
	}

	err = h.db.DeleteShoppingItemsByMealAndWeek(r.Context(), sqlc.DeleteShoppingItemsByMealAndWeekParams{
		Meal:          meal,
		WeekStartDate: weekStart,
	})

	if err != nil {
		log.Printf("Error deleting shopping items by meal: %v", err)
		http.Error(w, "Failed to delete items", http.StatusInternalServerError)
		return
	}

	// Redirect to refresh the page
	weekOffsetStr := r.URL.Query().Get("week_offset")
	weekOffset := 0
	if weekOffsetStr != "" {
		if offset, err := strconv.Atoi(weekOffsetStr); err == nil {
			weekOffset = offset
		}
	}

	http.Redirect(w, r, "/shopping?week="+strconv.Itoa(weekOffset), http.StatusSeeOther)
}

func (h *Handler) AutocompleteItems(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Add wildcards for LIKE query
	searchQuery := "%" + query + "%"

	items, err := h.db.SearchItemsAutocomplete(r.Context(), searchQuery)
	if err != nil {
		log.Printf("Error searching items: %v", err)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Format results
	type AutocompleteResult struct {
		Name  string `json:"name"`
		Meal  string `json:"meal,omitempty"`
		Count int64  `json:"count"`
	}

	results := make([]AutocompleteResult, 0, len(items))
	for _, item := range items {
		count := int64(0)
		if item.UseCount.Valid {
			count = item.UseCount.Int64
		}
		result := AutocompleteResult{
			Name:  item.Name,
			Count: count,
		}
		if item.Meal.Valid {
			result.Meal = item.Meal.String
		}
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) AutocompleteMeals(w http.ResponseWriter, r *http.Request) {
	// Get all meals from the meals table
	meals, err := h.db.ListMeals(r.Context())
	if err != nil {
		log.Printf("Error listing meals: %v", err)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	// Format results
	type AutocompleteResult struct {
		Meal string `json:"meal"`
	}

	results := make([]AutocompleteResult, 0, len(meals))
	for _, meal := range meals {
		results = append(results, AutocompleteResult{
			Meal: meal.Name,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
