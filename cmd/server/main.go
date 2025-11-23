package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
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

// runMigrations executes database migrations in a production-safe manner
func runMigrations(db *sql.DB) error {
	// Create migrations tracking table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Check if migration 1 needs to be auto-detected (for existing databases)
	// If the users table exists but migration 1 isn't tracked, mark it as applied
	var usersTableExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&usersTableExists)
	if err != nil {
		return fmt.Errorf("failed to check for existing tables: %w", err)
	}

	if usersTableExists > 0 {
		// Check if migration 1 is tracked
		var migration1Tracked int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 1").Scan(&migration1Tracked)
		if err != nil {
			return fmt.Errorf("failed to check migration tracking: %w", err)
		}

		if migration1Tracked == 0 {
			// Database has existing schema but no tracking - mark migration 1 as applied
			log.Printf("Detected existing schema from migration 1, marking as applied")
			_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (1)")
			if err != nil {
				return fmt.Errorf("failed to record existing migration 1: %w", err)
			}
		}
	}

	// Define available migrations
	migrations := []struct {
		version int
		file    string
	}{
		{1, "internal/database/migrations/001_initial_schema.sql"},
		{2, "internal/database/migrations/002_shopping_list.sql"},
		{3, "internal/database/migrations/003_meals_table.sql"},
	}

	// Check and run each migration
	for _, migration := range migrations {
		// Check if migration has already been applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", migration.version).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			log.Printf("Migration %d already applied, skipping", migration.version)
			continue
		}

		// Read migration file
		migrationSQL, err := os.ReadFile(migration.file)
		if err != nil {
			return fmt.Errorf("failed to read migration %d: %w", migration.version, err)
		}

		// Execute migration
		log.Printf("Applying migration %d...", migration.version)
		_, err = db.Exec(string(migrationSQL))
		if err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.version, err)
		}

		// Record migration as applied
		_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", migration.version)
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.version, err)
		}

		log.Printf("Migration %d applied successfully", migration.version)
	}

	return nil
}

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
	err = runMigrations(db)
	if err != nil {
		log.Fatal("Failed to run migrations:", err)
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

		r.Get("/shopping", h.ShoppingListView)
		r.Post("/shopping/items", h.CreateShoppingItem)
		r.Post("/shopping/items/{id}", h.UpdateShoppingItem)
		r.Delete("/shopping/items/{id}", h.DeleteShoppingItem)
		r.Post("/shopping/meals/{meal}/delete-all", h.DeleteShoppingItemsByMeal)
		r.Get("/shopping/meals/{meal}/quick-add", h.QuickAddMeal)
		r.Get("/shopping/autocomplete/items", h.AutocompleteItems)
		r.Get("/shopping/autocomplete/meals", h.AutocompleteMeals)

		r.Get("/meals", h.MealsList)
		r.Post("/meals", h.CreateMeal)
		r.Post("/meals/{id}", h.UpdateMeal)
		r.Delete("/meals/{id}", h.DeleteMeal)

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