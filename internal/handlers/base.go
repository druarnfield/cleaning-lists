package handlers

import (
	"github.com/a-h/templ"
	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"net/http"
)

type Handler struct {
	db   *sqlc.Queries
	auth *auth.AuthService
}

func NewHandler(db *sqlc.Queries, authService *auth.AuthService) *Handler {
	return &Handler{
		db:   db,
		auth: authService,
	}
}

// Render a templ component to the response writer
func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	return component.Render(r.Context(), w)
}