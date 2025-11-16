package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-co-op/gocron"
	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database"
	"github.com/druarnfield/cleaning-scheduler/internal/handlers"
	"github.com/druarnfield/cleaning-scheduler/internal/scheduler"
)

func main() {
	// Database setup
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "cleaning.db"
	}

	queries, db, err := database.NewDB(dbPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run migrations
	migrationSQL, err := os.ReadFile("internal/database/migrations/001_initial_schema.sql")
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		// Only log the error, don't fail - tables might already exist
		log.Printf("Migration notice: %v", err)
	}

	// Generate session secret if not exists
	sessionSecret := make([]byte, 32)
	rand.Read(sessionSecret)
	sessionSecretHex := hex.EncodeToString(sessionSecret)

	// Initialize services
	authService := auth.NewAuthService(queries, []byte(sessionSecretHex))
	h := handlers.NewHandler(queries, authService)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Static files
	fileServer := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public routes
	r.Group(func(r chi.Router) {
		r.Use(authService.RequireNoAuth)
		r.Get("/login", h.LoginPage)
		r.Post("/login", h.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authService.RequireAuth)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/schedule", http.StatusSeeOther)
		})

		r.Get("/setup-password", h.SetupPassword)
		r.Post("/setup-password", h.SetupPassword)

		r.Get("/schedule", h.ScheduleView)
		r.Post("/tasks/instances/{id}/toggle", h.ToggleCompletion)
		r.Post("/tasks/{id}/bring-forward", h.BringForwardTask)

		r.Get("/tasks", h.TasksList)
		r.Post("/tasks", h.CreateTask)
		r.Post("/tasks/{id}", h.UpdateTask)
		r.Delete("/tasks/{id}", h.DeleteTask)

		r.Get("/import", h.ImportCSV)
		r.Post("/import", h.ImportCSV)

		r.Get("/dashboard", h.DashboardView)

		r.Post("/logout", h.Logout)
		r.Get("/reset-password", h.ResetPasswordPage)
		r.Post("/reset-password", h.ResetPassword)
	})

	// Setup background scheduler
	s := gocron.NewScheduler(time.UTC)

	// Generate upcoming weeks every Sunday at midnight
	s.Every(1).Week().Sunday().At("00:00").Do(func() {
		log.Println("Running weekly task generation...")
		ctx := context.Background()
		err := scheduler.GenerateInstances(ctx, queries, time.Now(), 4)
		if err != nil {
			log.Printf("Error generating instances: %v", err)
		}
	})

	// Clean old data weekly
	s.Every(1).Week().Sunday().At("01:00").Do(func() {
		log.Println("Cleaning old data...")
		ctx := context.Background()
		queries.DeleteOldCompletions(ctx, sql.NullString{String: "90", Valid: true})
		queries.DeleteExpiredSessions(ctx)
	})

	// Start scheduler
	s.StartAsync()

	// Setup initial users if they don't exist
	ctx := context.Background()

	// Create dru user if doesn't exist
	_, err = queries.GetUser(ctx, "dru")
	if err == sql.ErrNoRows {
		log.Println("Creating user: dru")
		_, err = db.Exec(`INSERT INTO users (username) VALUES ('dru')`)
		if err != nil {
			log.Printf("Error creating dru user: %v", err)
		}
	}

	// Create josie user if doesn't exist
	_, err = queries.GetUser(ctx, "josie")
	if err == sql.ErrNoRows {
		log.Println("Creating user: josie")
		_, err = db.Exec(`INSERT INTO users (username) VALUES ('josie')`)
		if err != nil {
			log.Printf("Error creating josie user: %v", err)
		}
	}

	// Check if initial instances need to be generated
	count, _ := queries.CountInstances(ctx)
	if count == 0 {
		log.Println("Generating initial instances...")
		scheduler.GenerateInstances(ctx, queries, time.Now(), 4)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}