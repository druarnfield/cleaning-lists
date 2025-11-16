package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrPasswordNotSet     = errors.New("password not set")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidUsername    = errors.New("username must be 'dru' or 'josie'")
)

type AuthService struct {
	db    *sqlc.Queries
	store *sessions.CookieStore
}

type User struct {
	Username      string
	IsPasswordSet bool
}

func NewAuthService(db *sqlc.Queries, sessionSecret []byte) *AuthService {
	store := sessions.NewCookieStore(sessionSecret)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false, // Set to true in production with HTTPS
	}

	return &AuthService{
		db:    db,
		store: store,
	}
}

// Authenticate validates username and password
func (a *AuthService) Authenticate(ctx context.Context, username, password string) (*User, error) {
	// Normalize username to lowercase
	username = strings.ToLower(strings.TrimSpace(username))

	// Validate username
	if username != "dru" && username != "josie" {
		return nil, ErrInvalidUsername
	}

	// Get user from database
	user, err := a.db.GetUser(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Check if password is set
	if !user.IsPasswordSet.Bool {
		return &User{
			Username:      user.Username,
			IsPasswordSet: false,
		}, ErrPasswordNotSet
	}

	// Verify password
	if !user.PasswordHash.Valid || user.PasswordHash.String == "" {
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return &User{
		Username:      user.Username,
		IsPasswordSet: true,
	}, nil
}

// SetPassword sets a user's password for the first time
func (a *AuthService) SetPassword(ctx context.Context, username, newPassword string) error {
	// Normalize username
	username = strings.ToLower(strings.TrimSpace(username))

	// Validate username
	if username != "dru" && username != "josie" {
		return ErrInvalidUsername
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update user password
	return a.db.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		Username:     username,
		PasswordHash: sql.NullString{String: string(hashedPassword), Valid: true},
	})
}

// ResetPassword allows one user to reset another user's password
func (a *AuthService) ResetPassword(ctx context.Context, resetBy, targetUser, newPassword string) error {
	// Normalize usernames
	resetBy = strings.ToLower(strings.TrimSpace(resetBy))
	targetUser = strings.ToLower(strings.TrimSpace(targetUser))

	// Validate usernames
	if resetBy != "dru" && resetBy != "josie" {
		return fmt.Errorf("invalid reset by user: %s", resetBy)
	}
	if targetUser != "dru" && targetUser != "josie" {
		return fmt.Errorf("invalid target user: %s", targetUser)
	}

	// Can't reset own password through this method
	if resetBy == targetUser {
		return fmt.Errorf("cannot reset your own password")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Reset target user's password
	return a.db.ResetUserPassword(ctx, sqlc.ResetUserPasswordParams{
		Username:     targetUser,
		PasswordHash: sql.NullString{String: string(hashedPassword), Valid: true},
	})
}

// CreateSession creates a new session token for the user
func (a *AuthService) CreateSession(ctx context.Context, username string) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)

	// Calculate expiry (30 days from now)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	// Store in database
	err := a.db.CreateSession(ctx, sqlc.CreateSessionParams{
		Token:     token,
		Username:  username,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidateSession checks if a session token is valid
func (a *AuthService) ValidateSession(ctx context.Context, token string) (*User, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}

	// Get session from database
	session, err := a.db.GetSession(ctx, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	// Get user
	user, err := a.db.GetUser(ctx, session.Username)
	if err != nil {
		return nil, err
	}

	return &User{
		Username:      user.Username,
		IsPasswordSet: user.IsPasswordSet.Bool,
	}, nil
}

// DeleteSession removes a session token
func (a *AuthService) DeleteSession(ctx context.Context, token string) error {
	return a.db.DeleteSession(ctx, token)
}

// GetSessionFromRequest retrieves the session token from request cookies
func (a *AuthService) GetSessionFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetSessionCookie sets the session token cookie
func (a *AuthService) SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false, // Set to true in production with HTTPS
	})
}

// ClearSessionCookie clears the session token cookie
func (a *AuthService) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   false,
	})
}