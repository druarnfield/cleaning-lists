package handlers

import (
	"net/http"
	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	templAuth "github.com/druarnfield/cleaning-scheduler/internal/templates/auth"
)

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	component := templAuth.LoginPage("")
	Render(w, r, component)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.auth.Authenticate(r.Context(), username, password)
	if err == auth.ErrPasswordNotSet {
		// Create session for password setup
		token, _ := h.auth.CreateSession(r.Context(), username)
		h.auth.SetSessionCookie(w, token)
		http.Redirect(w, r, "/setup-password", http.StatusSeeOther)
		return
	}
	if err != nil {
		component := templAuth.LoginPage("Invalid username or password")
		Render(w, r, component)
		return
	}

	// Create session
	token, err := h.auth.CreateSession(r.Context(), user.Username)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	h.auth.SetSessionCookie(w, token)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) SetupPassword(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	if r.Method == http.MethodGet {
		// Check if password is already set
		if user.IsPasswordSet {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		component := templAuth.SetupPasswordPage(user, "")
		Render(w, r, component)
		return
	}

	// POST - set password
	newPassword := r.FormValue("new_password")
	confirmPassword := r.FormValue("confirm_password")

	if newPassword != confirmPassword {
		component := templAuth.SetupPasswordPage(user, "Passwords do not match")
		Render(w, r, component)
		return
	}

	if len(newPassword) < 6 {
		component := templAuth.SetupPasswordPage(user, "Password must be at least 6 characters")
		Render(w, r, component)
		return
	}

	err := h.auth.SetPassword(r.Context(), user.Username, newPassword)
	if err != nil {
		component := templAuth.SetupPasswordPage(user, "Failed to set password")
		Render(w, r, component)
		return
	}

	http.Redirect(w, r, "/schedule", http.StatusSeeOther)
}

func (h *Handler) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	component := templAuth.ResetPasswordPage(user, false, "")
	Render(w, r, component)
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := auth.GetUserFromContext(r.Context())
	targetUser := r.FormValue("target_user")
	newPassword := r.FormValue("new_password")

	// Can only reset the other user's password
	if targetUser == user.Username {
		component := templAuth.ResetPasswordPage(user, false, "Cannot reset your own password")
		Render(w, r, component)
		return
	}

	if len(newPassword) < 6 {
		component := templAuth.ResetPasswordPage(user, false, "Password must be at least 6 characters")
		Render(w, r, component)
		return
	}

	err := h.auth.SetPassword(r.Context(), targetUser, newPassword)
	if err != nil {
		component := templAuth.ResetPasswordPage(user, false, "Failed to reset password")
		Render(w, r, component)
		return
	}

	component := templAuth.ResetPasswordPage(user, true, "")
	Render(w, r, component)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := h.auth.GetSessionFromRequest(r)
	if token != "" {
		h.auth.DeleteSession(r.Context(), token)
	}
	h.auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}