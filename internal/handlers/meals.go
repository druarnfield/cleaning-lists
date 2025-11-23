package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	templPages "github.com/druarnfield/cleaning-scheduler/internal/templates/pages"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) MealsList(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	meals, err := h.db.ListMeals(r.Context())
	if err != nil {
		log.Printf("Error listing meals: %v", err)
		http.Error(w, "Failed to list meals", http.StatusInternalServerError)
		return
	}

	data := templPages.MealsPageData{
		Meals: meals,
	}

	component := templPages.MealsPage(user, data)
	Render(w, r, component)
}

func (h *Handler) CreateMeal(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))

	if name == "" {
		http.Error(w, "Meal name is required", http.StatusBadRequest)
		return
	}

	// Check if meal already exists
	_, err := h.db.GetMealByName(r.Context(), name)
	if err == nil {
		http.Error(w, "Meal with this name already exists", http.StatusConflict)
		return
	}

	_, err = h.db.CreateMeal(r.Context(), name)
	if err != nil {
		log.Printf("Error creating meal: %v", err)
		http.Error(w, "Failed to create meal", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/meals", http.StatusSeeOther)
}

func (h *Handler) UpdateMeal(w http.ResponseWriter, r *http.Request) {
	mealIDStr := chi.URLParam(r, "id")
	mealID, err := strconv.ParseInt(mealIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid meal ID", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "Meal name is required", http.StatusBadRequest)
		return
	}

	// Check if another meal with this name already exists
	existingMeal, err := h.db.GetMealByName(r.Context(), name)
	if err == nil && existingMeal.ID != mealID {
		http.Error(w, "Meal with this name already exists", http.StatusConflict)
		return
	}

	_, err = h.db.UpdateMeal(r.Context(), sqlc.UpdateMealParams{
		ID:   mealID,
		Name: name,
	})

	if err != nil {
		log.Printf("Error updating meal: %v", err)
		http.Error(w, "Failed to update meal", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") != "" {
		// HTMX request - return updated meal row
		meal, _ := h.db.GetMeal(r.Context(), mealID)
		component := templPages.MealRow(meal)
		Render(w, r, component)
	} else {
		http.Redirect(w, r, "/meals", http.StatusSeeOther)
	}
}

func (h *Handler) DeleteMeal(w http.ResponseWriter, r *http.Request) {
	mealIDStr := chi.URLParam(r, "id")
	mealID, err := strconv.ParseInt(mealIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid meal ID", http.StatusBadRequest)
		return
	}

	// Get meal name for usage check
	meal, err := h.db.GetMeal(r.Context(), mealID)
	if err != nil {
		http.Error(w, "Meal not found", http.StatusNotFound)
		return
	}

	// Check if meal is in use
	usageCount, err := h.db.CountMealUsage(r.Context(), meal.Name)
	if err != nil {
		log.Printf("Error checking meal usage: %v", err)
		http.Error(w, "Failed to check meal usage", http.StatusInternalServerError)
		return
	}

	if usageCount > 0 {
		http.Error(w, "Cannot delete meal that is in use by shopping items", http.StatusConflict)
		return
	}

	err = h.db.DeleteMeal(r.Context(), mealID)
	if err != nil {
		log.Printf("Error deleting meal: %v", err)
		http.Error(w, "Failed to delete meal", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") != "" {
		// HTMX request - return empty response to remove element
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/meals", http.StatusSeeOther)
	}
}
