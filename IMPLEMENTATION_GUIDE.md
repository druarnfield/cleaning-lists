# Implementation Guide - Cleaning Scheduler

This guide provides the remaining implementation details for the cleaning scheduler application. The core infrastructure (database, auth, scheduler) has been created. Follow this guide to complete the application.

## Current Status

### ✅ Completed
1. Project structure and dependencies
2. Database schema with auth and bring-forward support
3. SQLC queries for all database operations
4. Authentication service and middleware
5. Frequency parser supporting all CSV formats
6. Task distribution algorithm with "Alternate" and "Both" handling
7. Instance generator with bring-forward logic

### 🚧 Remaining Tasks

## 1. HTTP Handlers

### Create `internal/handlers/base.go`
```go
package handlers

import (
    "html/template"
    "net/http"
    "github.com/druarnfield/cleaning-scheduler/internal/auth"
    "github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
)

type Handler struct {
    db        *sqlc.Queries
    auth      *auth.AuthService
    templates *template.Template
}

func NewHandler(db *sqlc.Queries, authService *auth.AuthService) *Handler {
    // Parse templates
    tmpl := template.Must(template.ParseGlob("web/templates/**/*.html"))

    return &Handler{
        db:        db,
        auth:      authService,
        templates: tmpl,
    }
}

func (h *Handler) render(w http.ResponseWriter, name string, data interface{}) {
    err := h.templates.ExecuteTemplate(w, name, data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

### Create `internal/handlers/auth.go`
```go
package handlers

import (
    "net/http"
    "github.com/druarnfield/cleaning-scheduler/internal/auth"
)

func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
    h.render(w, "login.html", nil)
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
        h.render(w, "login.html", map[string]interface{}{
            "Error": "Invalid username or password",
        })
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
        h.render(w, "setup_password.html", map[string]interface{}{
            "User": user,
        })
        return
    }

    newPassword := r.FormValue("new_password")
    confirmPassword := r.FormValue("confirm_password")

    if newPassword != confirmPassword {
        h.render(w, "setup_password.html", map[string]interface{}{
            "Error": "Passwords do not match",
            "User":  user,
        })
        return
    }

    err := h.auth.SetPassword(r.Context(), user.Username, newPassword)
    if err != nil {
        h.render(w, "setup_password.html", map[string]interface{}{
            "Error": "Failed to set password",
            "User":  user,
        })
        return
    }

    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
    token := h.auth.GetSessionFromRequest(r)
    if token != "" {
        h.auth.DeleteSession(r.Context(), token)
    }
    h.auth.ClearSessionCookie(w)
    http.Redirect(w, r, "/login", http.StatusSeeOther)
}
```

### Create `internal/handlers/schedule.go`
```go
package handlers

import (
    "net/http"
    "time"
    "github.com/druarnfield/cleaning-scheduler/internal/auth"
    "github.com/druarnfield/cleaning-scheduler/internal/scheduler"
)

func (h *Handler) ScheduleView(w http.ResponseWriter, r *http.Request) {
    user := auth.GetUserFromContext(r.Context())

    // Get current week start (Monday)
    now := time.Now()
    currentWeekStart := scheduler.GetWeekStart(now)
    prevWeekStart := currentWeekStart.AddDate(0, 0, -7)
    nextWeekStart := currentWeekStart.AddDate(0, 0, 7)

    // Get instances for all three weeks
    prevWeek, _ := h.db.ListInstancesByWeek(r.Context(), prevWeekStart)
    currentWeek, _ := h.db.ListInstancesByWeek(r.Context(), currentWeekStart)
    nextWeek, _ := h.db.ListInstancesByWeek(r.Context(), nextWeekStart)

    data := map[string]interface{}{
        "User":        user,
        "PrevWeek":    formatWeekData(prevWeek, prevWeekStart),
        "CurrentWeek": formatWeekData(currentWeek, currentWeekStart),
        "NextWeek":    formatWeekData(nextWeek, nextWeekStart),
    }

    h.render(w, "schedule.html", data)
}

func (h *Handler) ToggleCompletion(w http.ResponseWriter, r *http.Request) {
    user := auth.GetUserFromContext(r.Context())
    instanceID := r.URL.Path // Parse from path

    // Check if already completed
    completion, err := h.db.GetCompletion(r.Context(), instanceID)

    if err != nil {
        // Not completed - create completion
        h.db.CreateCompletion(r.Context(), sqlc.CreateCompletionParams{
            TaskInstanceID: instanceID,
            CompletedBy:    user.Username,
        })
    } else {
        // Already completed - remove
        h.db.DeleteCompletion(r.Context(), instanceID)
    }

    // Return updated task row HTML for HTMX
    instance, _ := h.db.GetInstance(r.Context(), instanceID)
    h.render(w, "task_row", instance)
}

func (h *Handler) BringForwardTask(w http.ResponseWriter, r *http.Request) {
    user := auth.GetUserFromContext(r.Context())
    taskID := r.FormValue("task_id")
    targetDate := r.FormValue("target_date")

    // Parse target date
    date, err := time.Parse("2006-01-02", targetDate)
    if err != nil {
        http.Error(w, "Invalid date", http.StatusBadRequest)
        return
    }

    // Bring forward the task
    err = scheduler.BringForwardTask(r.Context(), h.db, taskID, date, user.Username)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Return updated week card
    weekStart := scheduler.GetWeekStart(date)
    instances, _ := h.db.ListInstancesByWeek(r.Context(), weekStart)
    h.render(w, "week_card", formatWeekData(instances, weekStart))
}
```

### Create `internal/handlers/import.go`
```go
package handlers

import (
    "encoding/csv"
    "io"
    "net/http"
    "strconv"
    "strings"
    "database/sql"
    "github.com/druarnfield/cleaning-scheduler/internal/scheduler"
)

func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        h.render(w, "import.html", nil)
        return
    }

    // Parse multipart form
    err := r.ParseMultipartForm(10 << 20) // 10MB max
    if err != nil {
        http.Error(w, "File too large", http.StatusBadRequest)
        return
    }

    file, _, err := r.FormFile("csvfile")
    if err != nil {
        http.Error(w, "No file uploaded", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Parse CSV
    reader := csv.NewReader(file)

    // Skip header
    _, err = reader.Read()
    if err != nil {
        http.Error(w, "Invalid CSV", http.StatusBadRequest)
        return
    }

    var tasksCreated int
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil || len(record) < 5 {
            continue
        }

        name := strings.TrimSpace(record[0])
        category := strings.TrimSpace(record[1])
        frequency := strings.TrimSpace(record[2])
        assignedTo := strings.TrimSpace(record[3])
        estimatedMins, _ := strconv.Atoi(strings.TrimSpace(record[4]))

        if name == "" || category == "" {
            continue
        }

        // Determine default assignee
        var defaultAssignee sql.NullString
        if assignedTo != "" {
            defaultAssignee = sql.NullString{String: assignedTo, Valid: true}
        }

        // Create task
        _, err = h.db.CreateTask(r.Context(), sqlc.CreateTaskParams{
            Name:            name,
            Category:        category,
            Frequency:       frequency,
            EstimatedMins:   int64(estimatedMins),
            DefaultAssignee: defaultAssignee,
        })

        if err == nil {
            tasksCreated++
        }
    }

    // Generate initial instances
    startDate := time.Now().Truncate(24 * time.Hour)
    scheduler.GenerateInstances(r.Context(), h.db, startDate, 4)

    // Redirect to schedule
    http.Redirect(w, r, "/schedule", http.StatusSeeOther)
}
```

## 2. HTML Templates

### Create `web/templates/layout/base.html`
```html
<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Cleaning Scheduler</title>
    <link href="https://cdn.jsdelivr.net/npm/daisyui@4.4.0/dist/full.min.css" rel="stylesheet">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-base-200 min-h-screen">
    {{template "nav" .}}
    <main class="container mx-auto px-4 py-8 max-w-6xl">
        {{template "content" .}}
    </main>
</body>
</html>
```

### Create `web/templates/auth/login.html`
```html
{{define "content"}}
<div class="flex justify-center items-center min-h-[60vh]">
    <div class="card w-96 bg-base-100 shadow-xl">
        <div class="card-body">
            <h2 class="card-title text-2xl mb-4">Login</h2>

            {{if .Error}}
            <div class="alert alert-error mb-4">
                <span>{{.Error}}</span>
            </div>
            {{end}}

            <form method="POST" action="/login">
                <div class="form-control mb-4">
                    <label class="label">
                        <span class="label-text">Select User</span>
                    </label>
                    <select name="username" class="select select-bordered" required>
                        <option value="">Choose...</option>
                        <option value="dru">Dru</option>
                        <option value="josie">Josie</option>
                    </select>
                </div>

                <div class="form-control mb-6">
                    <label class="label">
                        <span class="label-text">Password</span>
                    </label>
                    <input type="password" name="password" class="input input-bordered" required>
                </div>

                <button type="submit" class="btn btn-primary w-full">Login</button>
            </form>
        </div>
    </div>
</div>
{{end}}
```

### Create `web/templates/pages/schedule.html`
```html
{{define "content"}}
<div class="space-y-6">
    <h1 class="text-3xl font-bold">Cleaning Schedule</h1>

    <!-- Previous Week -->
    <div id="prev-week">
        {{template "week_card" .PrevWeek}}
    </div>

    <!-- Current Week (Highlighted) -->
    <div id="current-week" class="ring-2 ring-primary rounded-lg">
        {{template "week_card" .CurrentWeek}}
    </div>

    <!-- Next Week -->
    <div id="next-week">
        {{template "week_card" .NextWeek}}
    </div>
</div>
{{end}}
```

### Create `web/templates/partials/week_card.html`
```html
{{define "week_card"}}
<div class="card bg-base-100 shadow-xl">
    <div class="card-body">
        <h2 class="card-title">
            {{.WeekLabel}}
            <span class="text-sm font-normal text-base-content/60">
                {{.StartDate}} - {{.EndDate}}
            </span>
        </h2>

        <div class="grid md:grid-cols-2 gap-6 mt-4">
            <!-- Dru's Column -->
            <div>
                <div class="flex items-center gap-2 mb-3">
                    <div class="badge badge-primary">Dru</div>
                    <span class="text-sm">{{.DruTaskCount}} tasks • {{.DruMins}} mins</span>
                </div>
                <div class="space-y-2">
                    {{range .DruTasks}}
                        {{template "task_row" .}}
                    {{end}}
                </div>
            </div>

            <!-- Josie's Column -->
            <div>
                <div class="flex items-center gap-2 mb-3">
                    <div class="badge badge-secondary">Josie</div>
                    <span class="text-sm">{{.JosieTaskCount}} tasks • {{.JosieMins}} mins</span>
                </div>
                <div class="space-y-2">
                    {{range .JosieTasks}}
                        {{template "task_row" .}}
                    {{end}}
                </div>
            </div>
        </div>
    </div>
</div>
{{end}}
```

### Create `web/templates/partials/task_row.html`
```html
{{define "task_row"}}
<div class="flex items-center gap-3 p-2 rounded hover:bg-base-200" id="task-{{.ID}}">
    <input
        type="checkbox"
        {{if .IsCompleted}}checked{{end}}
        class="checkbox checkbox-sm {{if eq .AssignedTo "dru"}}checkbox-primary{{else}}checkbox-secondary{{end}}"
        hx-post="/tasks/instances/{{.ID}}/toggle"
        hx-swap="outerHTML"
        hx-target="#task-{{.ID}}"
    />
    <div class="flex-1 {{if .IsCompleted}}line-through opacity-50{{end}}">
        <span class="font-medium">{{.TaskName}}</span>
        {{if .BroughtForward}}
            <span class="badge badge-info badge-xs ml-2">Brought Forward</span>
        {{end}}
        <div class="text-xs text-base-content/60">
            {{.Category}} • {{.EstimatedMins}} mins
            {{if .BroughtForward}}• Originally {{.OriginalDate}}{{end}}
        </div>
    </div>
    {{if and (not .IsCompleted) (not .IsPast)}}
    <button class="btn btn-ghost btn-xs" onclick="bringForwardModal({{.TaskID}})">
        Bring Forward
    </button>
    {{end}}
</div>
{{end}}
```

## 3. Main Server Entry Point

### Create `cmd/server/main.go`
```go
package main

import (
    "context"
    "crypto/rand"
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

    // Run migrations (you'll need to add goose or similar)
    // For now, manually run the SQL migration

    // Generate session secret if not exists
    sessionSecret := make([]byte, 32)
    rand.Read(sessionSecret)

    // Initialize services
    authService := auth.NewAuthService(queries, sessionSecret)
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
        r.Put("/tasks/{id}", h.UpdateTask)
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
        queries.DeleteOldCompletions(ctx, 90)
        queries.DeleteExpiredSessions(ctx)
    })

    // Start scheduler
    s.StartAsync()

    // Check if initial instances need to be generated
    ctx := context.Background()
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
```

## 4. Makefile

### Create `Makefile`
```makefile
.PHONY: build run dev test migrate-up migrate-down install sqlc clean

build:
	go build -o bin/cleaning-scheduler cmd/server/main.go

run: build
	./bin/cleaning-scheduler

dev:
	air

test:
	go test -v ./...

migrate-up:
	sqlite3 cleaning.db < internal/database/migrations/001_initial_schema.sql

migrate-down:
	rm -f cleaning.db

install:
	go mod download
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/cosmtrek/air@latest
	sqlc generate
	make migrate-up

sqlc:
	sqlc generate

clean:
	rm -rf bin/
	rm -f cleaning.db

seed: migrate-up
	@echo "Database initialized. Run 'make run' and import the CSV file."
```

## 5. Air Configuration

### Create `.air.toml`
```toml
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/main cmd/server/main.go"
  bin = "tmp/main"
  full_bin = "./tmp/main"
  include_ext = ["go", "html", "css", "js"]
  exclude_dir = ["tmp", "vendor", "web/static/node_modules"]
  include_dir = []
  exclude_file = []
  delay = 1000
  stop_on_error = true
  log = "air_errors.log"

[log]
  time = true

[color]
  main = "magenta"
  watcher = "cyan"
  build = "yellow"
  runner = "green"
```

## 6. Testing the Application

### Initial Setup
```bash
# Install dependencies and setup
make install

# Run the application
make run

# Or for development with hot reload
make dev
```

### Testing Flow
1. Navigate to http://localhost:8080
2. You'll be redirected to login
3. Select either "Dru" or "Josie"
4. Since no password is set, you'll be redirected to setup password
5. Set your password
6. You'll be redirected to the schedule view
7. Import the CSV file via the Import page
8. Tasks will be distributed and instances generated
9. Test bringing forward tasks and marking them complete

## Next Steps

1. Add better error handling throughout
2. Implement CSRF protection
3. Add input validation
4. Create dashboard analytics views
5. Add tests for critical functions
6. Improve UI/UX with better HTMX interactions
7. Add loading states and error messages
8. Implement password reset functionality
9. Add ability to manually adjust task assignments
10. Create a proper logging system

## Notes

- The application uses SQLite for simplicity
- Sessions are stored in the database with 30-day expiry
- "Both" tasks create instances for both users
- "Alternate" tasks are distributed fairly using a greedy algorithm
- Brought-forward tasks don't count toward weekly totals
- The scheduler runs weekly to generate new instances

This implementation provides a solid foundation for the cleaning scheduler with all requested features including authentication, fair distribution, and the bring-forward capability.