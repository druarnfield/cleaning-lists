package auth

import (
	"context"
	"net/http"
)

type contextKey string

const UserContextKey contextKey = "user"

// RequireAuth is middleware that checks if the user is authenticated
func (a *AuthService) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session token from cookie
		token := a.GetSessionFromRequest(r)
		if token == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Validate session
		user, err := a.ValidateSession(r.Context(), token)
		if err != nil {
			a.ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Check if password is set (for first-time users)
		if !user.IsPasswordSet && r.URL.Path != "/setup-password" {
			http.Redirect(w, r, "/setup-password", http.StatusSeeOther)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves the user from request context
func GetUserFromContext(ctx context.Context) *User {
	user, ok := ctx.Value(UserContextKey).(*User)
	if !ok {
		return nil
	}
	return user
}

// RequireNoAuth is middleware for pages that should only be accessible to non-authenticated users
func (a *AuthService) RequireNoAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session token from cookie
		token := a.GetSessionFromRequest(r)
		if token != "" {
			// Validate session
			_, err := a.ValidateSession(r.Context(), token)
			if err == nil {
				// User is authenticated, redirect to home
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}