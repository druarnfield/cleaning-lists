# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a household cleaning task scheduler web application for two people (Dru and Josie) that automatically distributes recurring cleaning tasks fairly based on time commitment. The project specification is detailed in `CLEANING_SCHEDULER_PROJECT_SPEC.md`.

## Technology Stack

- **Backend**: Go 1.21+ with Chi v5 router
- **Database**: SQLite 3 with sqlc for type-safe queries
- **Frontend**: HTMX 1.9+ with TailwindCSS/DaisyUI (server-side rendering)
- **Background Jobs**: gocron for scheduling
- **Development**: Air for hot reloading

## Common Commands

### Initial Setup
```bash
# Install dependencies and tools
go mod init github.com/yourusername/cleaning-scheduler
go get github.com/go-chi/chi/v5
go get github.com/mattn/go-sqlite3
go get github.com/go-co-op/gocron
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/cosmtrek/air@latest

# Generate sqlc code (after writing queries)
sqlc generate

# Run database migrations
goose -dir internal/database/migrations sqlite3 cleaning.db up
```

### Development
```bash
# Run with hot reload
air

# Build binary
go build -o bin/cleaning-scheduler cmd/server/main.go

# Run server
./bin/cleaning-scheduler

# Run tests
go test -v ./...

# Run specific test
go test -v ./internal/scheduler -run TestParseFrequency
```

### Database Operations
```bash
# Apply migrations
goose -dir internal/database/migrations sqlite3 cleaning.db up

# Rollback migrations
goose -dir internal/database/migrations sqlite3 cleaning.db down

# Seed sample data
sqlite3 cleaning.db < scripts/seed.sql
```

## Code Architecture

### Project Structure
The codebase follows standard Go project layout with clear separation of concerns:

- **cmd/server/main.go**: Application entry point, router setup, background scheduler initialization
- **internal/database/**: Database layer with migrations, sqlc queries, and connection management
- **internal/handlers/**: HTTP handlers for all endpoints (schedule, tasks, completion, dashboard, import)
- **internal/scheduler/**: Core business logic for task distribution, instance generation, and frequency parsing
- **internal/analytics/**: Statistics and analytics calculations
- **internal/models/**: Domain models and DTOs
- **web/templates/**: HTML templates (base layout, pages, HTMX partials)
- **web/static/**: CSS and minimal JavaScript

### Key Business Logic

#### Fair Task Distribution Algorithm
Located in `internal/scheduler/distributor.go`. Uses a greedy algorithm to distribute unassigned tasks between Dru and Josie based on total weekly time commitment. Pre-assigned tasks are respected.

#### Task Instance Generation
Located in `internal/scheduler/generator.go`. Generates recurring task instances for the next 4 weeks, with randomized first occurrences for variety. Runs weekly via background scheduler.

#### Frequency Parsing
Located in `internal/scheduler/frequency.go`. Parses various frequency formats:
- Standard: Daily, Weekly, Fortnightly, Monthly
- Custom: "N Weekly" (e.g., "12 Weekly"), "N x/week" (e.g., "2x/week")

#### HTMX Integration
All interactive elements use HTMX for server-driven UI updates without full page reloads:
- Task completion toggle returns updated task row HTML
- Week navigation swaps week cards
- Form submissions can return partials or redirects

### Database Schema
Three main tables with appropriate indexes:
- **tasks**: Core task definitions (name, category, frequency, estimated time, default assignee)
- **task_instances**: Scheduled occurrences with assignments and week associations
- **completions**: Tracks when instances are marked complete

## Development Guidelines

### When Adding Features
1. Follow existing patterns in handlers and templates
2. Use sqlc for all database queries - add to `internal/database/queries/`
3. Write tests for business logic (scheduler, analytics, frequency parsing)
4. Ensure HTMX interactions return appropriate partials
5. Maintain responsive design with Tailwind/DaisyUI classes

### Testing Strategy
- Unit test coverage targets: Scheduler (90%+), Analytics (85%+), Handlers (70%+)
- Integration tests for key user flows
- Manual testing checklist in spec document

### Key Implementation Details
- Week starts on Monday for scheduling purposes
- Background scheduler runs Sunday at midnight to generate next 4 weeks
- CSV import format: Task,Category,Frequency,Assigned To,Estimated Mins
- All times stored in UTC, displayed in local timezone
- Task completion is a toggle operation (can mark/unmark as complete)

## Important Notes

- The application is designed for a single household (no multi-tenancy)
- SQLite database file (`cleaning.db`) contains all persistent data
- No external API dependencies - fully self-contained
- HTMX enables dynamic updates without writing JavaScript
- Fair distribution algorithm runs after CSV import or manual rebalance