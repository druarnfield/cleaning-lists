# Household Cleaning Task Scheduler - Project Specification

**Version:** 1.0  
**Last Updated:** November 16, 2025  
**Target Stack:** Go + Chi + HTMX + SQLite

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Project Overview](#project-overview)
3. [Technical Architecture](#technical-architecture)
4. [Database Design](#database-design)
5. [API Endpoints](#api-endpoints)
6. [User Interface Specifications](#user-interface-specifications)
7. [Business Logic](#business-logic)
8. [Implementation Plan](#implementation-plan)
9. [Testing Strategy](#testing-strategy)
10. [Deployment](#deployment)
11. [Future Enhancements](#future-enhancements)

---

## Executive Summary

A web-based household cleaning task scheduler for two people (Dru and Josie) that automatically distributes recurring cleaning tasks fairly based on time commitment, generates weekly schedules, tracks completions, and provides analytics on task completion over time.

**Key Features:**
- Fair automatic task distribution based on estimated time
- Weekly schedule view with completion tracking
- Task management (CRUD operations)
- Analytics dashboard showing completion rates and workload balance
- Background generation of upcoming task instances
- CSV import for initial task setup

**Technical Highlights:**
- Single binary deployment
- No external dependencies beyond SQLite
- Fast, responsive HTMX-driven UI
- Type-safe database queries with sqlc
- Persistent storage with full history

---

## Project Overview

### Problem Statement

Two people sharing a household need to fairly distribute recurring cleaning tasks with varying frequencies (daily, weekly, monthly, etc.) and time commitments. Manual scheduling is time-consuming and often results in perceived unfairness in workload distribution.

### Solution

An automated web application that:
1. Reads task definitions (name, frequency, estimated time)
2. Fairly distributes tasks between two people based on total time commitment
3. Generates a rolling schedule showing upcoming weeks
4. Tracks task completion with persistent storage
5. Provides analytics on completion rates and workload balance

### Target Users

- Primary: Dru and Josie (household members)
- Deployment: Local network (single household)
- Access: Web browser on desktop/mobile devices

### Core Requirements

**Must Have (MVP):**
- View current, previous, and next week's tasks
- Check off completed tasks (persists across sessions)
- Add/edit/delete tasks via UI
- Fair automatic task distribution
- Import tasks from CSV file
- Dashboard with completion statistics

**Should Have (Phase 2):**
- Reassign specific task instances
- Filter/search tasks by category
- Export completion history
- Manual rebalancing of workload

**Nice to Have (Future):**
- Mobile notifications
- Calendar sync
- Multiple households
- Task notes/comments

---

## Technical Architecture

### Technology Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Backend Language | Go 1.21+ | Performance, single binary, strong stdlib |
| Web Framework | Chi v5 | Lightweight, idiomatic Go router |
| Database | SQLite 3 | Embedded, zero-config, perfect for single household |
| Query Builder | sqlc | Type-safe SQL queries generated from SQL |
| Frontend | HTMX 1.9+ | Minimal JS, server-driven UI updates |
| Styling | TailwindCSS + DaisyUI | Rapid UI development, consistent design |
| Templating | Go html/template | Standard library, adequate for this use case |
| Scheduler | gocron | Simple cron-like scheduling in Go |
| Dev Tooling | Air | Hot reloading during development |

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Schedule   │  │     Tasks    │  │   Dashboard  │      │
│  │     View     │  │  Management  │  │   Analytics  │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │              │
│         └──────────────────┴──────────────────┘              │
│                           │                                  │
│                      HTMX Requests                           │
└───────────────────────────┼──────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Go HTTP Server                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                   Chi Router                          │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │   │
│  │  │  Schedule   │  │    Task     │  │  Dashboard  │  │   │
│  │  │  Handlers   │  │  Handlers   │  │  Handlers   │  │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  │   │
│  └─────────┼────────────────┼────────────────┼─────────┘   │
│            └────────────────┴────────────────┘              │
│                            │                                 │
│  ┌─────────────────────────┼──────────────────────────┐    │
│  │              Business Logic Layer                   │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │    │
│  │  │   Scheduler  │  │ Distributor  │  │Analytics │ │    │
│  │  │  (Generate   │  │ (Fair Task   │  │  Engine  │ │    │
│  │  │  Instances)  │  │ Assignment)  │  │          │ │    │
│  │  └──────┬───────┘  └──────┬───────┘  └────┬─────┘ │    │
│  └─────────┼──────────────────┼───────────────┼───────┘    │
│            └──────────────────┴───────────────┘             │
│                            │                                 │
│  ┌─────────────────────────┼──────────────────────────┐    │
│  │                 Data Access Layer (sqlc)            │    │
│  │         Type-safe queries generated from SQL        │    │
│  └─────────────────────────┼──────────────────────────┘    │
└────────────────────────────┼─────────────────────────────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │  SQLite Database │
                    │   cleaning.db    │
                    └──────────────────┘

Background Process (gocron):
┌─────────────────────────────────┐
│  Weekly Scheduler (Sunday 00:00) │
│  • Generate next 4 weeks         │
│  • Clean old completions (90d+)  │
└─────────────────────────────────┘
```

### Project Directory Structure

```
cleaning-scheduler/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
│
├── internal/
│   ├── database/
│   │   ├── migrations/             # SQL migration files
│   │   │   ├── 001_initial_schema.sql
│   │   │   ├── 002_add_indexes.sql
│   │   │   └── ...
│   │   ├── queries/                # sqlc query definitions
│   │   │   ├── tasks.sql
│   │   │   ├── instances.sql
│   │   │   ├── completions.sql
│   │   │   └── analytics.sql
│   │   ├── sqlc/                   # Generated sqlc code (gitignore)
│   │   │   ├── db.go
│   │   │   ├── models.go
│   │   │   └── querier.go
│   │   └── db.go                   # Database connection & setup
│   │
│   ├── handlers/
│   │   ├── schedule.go             # Schedule view handlers
│   │   ├── tasks.go                # Task CRUD handlers
│   │   ├── completion.go           # Toggle completion
│   │   ├── dashboard.go            # Analytics/stats
│   │   ├── import.go               # CSV import
│   │   └── helpers.go              # Shared handler utilities
│   │
│   ├── scheduler/
│   │   ├── generator.go            # Task instance generation
│   │   ├── distributor.go          # Fair workload distribution
│   │   └── frequency.go            # Frequency parsing logic
│   │
│   ├── analytics/
│   │   └── stats.go                # Completion rate calculations
│   │
│   ├── models/
│   │   └── models.go               # Domain models/DTOs
│   │
│   └── middleware/
│       └── logger.go               # HTTP request logging
│
├── web/
│   ├── templates/
│   │   ├── layout/
│   │   │   ├── base.html           # Base HTML structure
│   │   │   └── nav.html            # Navigation component
│   │   ├── pages/
│   │   │   ├── schedule.html       # Main schedule view
│   │   │   ├── tasks.html          # Task management page
│   │   │   ├── dashboard.html      # Analytics dashboard
│   │   │   └── import.html         # CSV import page
│   │   └── partials/               # HTMX partial templates
│   │       ├── week-card.html      # Single week display
│   │       ├── task-row.html       # Task checkbox row
│   │       ├── task-form.html      # Add/edit task form
│   │       ├── stats-widget.html   # Dashboard stats
│   │       └── task-list.html      # Task management list
│   │
│   └── static/
│       ├── css/
│       │   └── styles.css          # Custom styles (if needed)
│       └── js/
│           └── app.js              # Minimal JS for HTMX enhancements
│
├── scripts/
│   └── seed.sh                     # Seed database with sample data
│
├── .air.toml                       # Air hot reload config
├── .gitignore
├── go.mod
├── go.sum
├── Makefile                        # Build, run, migrate commands
├── README.md
└── sqlc.yaml                       # sqlc configuration
```

### Data Flow

#### 1. Initial Setup Flow
```
User uploads CSV
    ↓
Parse CSV → Validate → Insert tasks into DB
    ↓
Calculate fair distribution (Distributor)
    ↓
Generate initial 4 weeks of instances (Scheduler)
    ↓
Redirect to schedule view
```

#### 2. Weekly Schedule View Flow
```
User navigates to / or /schedule
    ↓
Get current date → Determine current week (Monday-Sunday)
    ↓
Query instances for previous, current, next week
    ↓
Fetch completion status for each instance
    ↓
Render 3-week view with HTMX-enabled checkboxes
```

#### 3. Task Completion Flow (HTMX)
```
User clicks checkbox
    ↓
HTMX POST to /tasks/instances/{id}/toggle
    ↓
Handler: Toggle completion in DB (insert or delete)
    ↓
Return updated task-row.html partial
    ↓
HTMX swaps checkbox HTML (strikethrough, opacity change)
```

#### 4. Background Scheduler Flow
```
Cron job runs (Sunday 00:00)
    ↓
Check if upcoming weeks exist
    ↓
Generate missing weeks (always maintain 4 weeks ahead)
    ↓
Clean up old completions (>90 days)
```

---

## Database Design

### Schema Overview

```sql
-- Core task definitions (long-lived, user editable)
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    frequency TEXT NOT NULL,  -- 'Daily', 'Weekly', 'Fortnightly', 'Monthly', '2 Weekly', etc.
    estimated_mins INTEGER NOT NULL,
    default_assignee TEXT,    -- 'Dru', 'Josie', or NULL for auto-assign
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Scheduled task instances (generated weekly, represents specific occurrences)
CREATE TABLE task_instances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    scheduled_date DATE NOT NULL,
    assigned_to TEXT NOT NULL,  -- 'Dru' or 'Josie'
    week_start_date DATE NOT NULL,  -- Monday of the week this task falls in
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Task completions (tracks when tasks were marked done)
CREATE TABLE completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_instance_id INTEGER NOT NULL,
    completed_by TEXT NOT NULL,  -- 'Dru' or 'Josie'
    completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    actual_mins INTEGER,  -- Optional: if user wants to track actual time
    FOREIGN KEY (task_instance_id) REFERENCES task_instances(id) ON DELETE CASCADE,
    UNIQUE(task_instance_id)  -- Each instance can only be completed once
);

-- Application settings
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_instances_week ON task_instances(week_start_date);
CREATE INDEX idx_instances_date ON task_instances(scheduled_date);
CREATE INDEX idx_instances_assigned ON task_instances(assigned_to);
CREATE INDEX idx_completions_instance ON completions(task_instance_id);
CREATE INDEX idx_completions_date ON completions(completed_at);
```

### Entity Relationships

```
┌─────────────┐
│    tasks    │
│             │
│ id          │◄─────┐
│ name        │      │
│ category    │      │
│ frequency   │      │
│ est_mins    │      │
│ assignee    │      │
└─────────────┘      │
                     │
                     │ 1:N
                     │
              ┌──────┴──────────┐
              │ task_instances  │
              │                 │
              │ id              │◄─────┐
              │ task_id         │      │
              │ scheduled_date  │      │
              │ assigned_to     │      │
              │ week_start_date │      │
              └─────────────────┘      │
                                       │ 1:1
                                       │
                                ┌──────┴─────────┐
                                │  completions   │
                                │                │
                                │ id             │
                                │ task_inst_id   │
                                │ completed_by   │
                                │ completed_at   │
                                │ actual_mins    │
                                └────────────────┘
```

### Initial Data Seeds

**Settings Table:**
```sql
INSERT INTO settings (key, value) VALUES
    ('start_date', '2025-11-16'),
    ('weeks_ahead', '4'),
    ('app_version', '1.0.0');
```

### Sample Data (for testing)

```sql
-- Sample tasks
INSERT INTO tasks (name, category, frequency, estimated_mins, default_assignee) VALUES
    ('General Bathroom', 'Bathroom', 'Weekly', 30, NULL),
    ('Clean Bathtub', 'Bathroom', 'Monthly', 30, NULL),
    ('Josie Laundry', 'Laundry', 'Weekly', 30, 'Josie'),
    ('Dru Laundry', 'Laundry', 'Weekly', 30, 'Dru'),
    ('Empty Fridge', 'Kitchen', 'Weekly', 15, NULL),
    ('Deep Kitchen Clean', 'Kitchen', '12 Weekly', 90, NULL),
    ('Mop Kitchen', 'Kitchen', 'Monthly', 30, NULL);
```

---

## API Endpoints

### REST-like Routes (returning HTML)

| Method | Path | Description | Returns |
|--------|------|-------------|---------|
| GET | `/` | Redirect to schedule | 302 → /schedule |
| GET | `/schedule` | Main 3-week schedule view | Full HTML page |
| GET | `/schedule/week/{date}` | Single week card (HTMX) | Partial HTML |
| GET | `/tasks` | Task management page | Full HTML page |
| GET | `/tasks/new` | New task form | Partial HTML (modal) |
| GET | `/tasks/{id}/edit` | Edit task form | Partial HTML (modal) |
| POST | `/tasks` | Create new task | Redirect or partial |
| PUT | `/tasks/{id}` | Update task | Redirect or partial |
| DELETE | `/tasks/{id}` | Delete task | HTMX OOB swap |
| POST | `/tasks/import` | Import CSV file | Redirect to schedule |
| POST | `/tasks/instances/{id}/toggle` | Toggle completion | Partial HTML (task row) |
| GET | `/dashboard` | Analytics dashboard | Full HTML page |
| GET | `/dashboard/stats/{period}` | Refresh stats widget | Partial HTML |
| POST | `/tasks/rebalance` | Recalculate fair distribution | Redirect to tasks |

### Request/Response Examples

#### Toggle Task Completion (HTMX)

**Request:**
```http
POST /tasks/instances/42/toggle
Content-Type: application/x-www-form-urlencoded
HX-Request: true

```

**Response:**
```html
<div class="task-row" hx-swap-oob="true" id="task-42">
    <label class="cursor-pointer label">
        <input 
            type="checkbox" 
            checked
            class="checkbox checkbox-primary"
            hx-post="/tasks/instances/42/toggle"
            hx-swap="outerHTML"
            hx-target="closest .task-row"
        />
        <span class="label-text ml-3 line-through opacity-50">
            General Bathroom (30 mins)
        </span>
    </label>
</div>
```

#### Create Task

**Request:**
```http
POST /tasks
Content-Type: application/x-www-form-urlencoded

name=Vacuum+Bedroom&category=Bedroom&frequency=Weekly&estimated_mins=20&default_assignee=
```

**Response:**
```http
HTTP/1.1 303 See Other
Location: /tasks
HX-Redirect: /tasks
```

---

## User Interface Specifications

### Design System

**Framework:** DaisyUI (Tailwind CSS component library)  
**Theme:** Light theme with subtle colors  
**Typography:** System font stack  
**Color Scheme:**
- Primary: Blue (Dru's tasks)
- Secondary: Pink/Purple (Josie's tasks)
- Neutral: Gray tones
- Success: Green (completed tasks)

### Page Layouts

#### 1. Schedule View (`/schedule`)

**Layout:**
```
┌────────────────────────────────────────────────────┐
│  Header: "Cleaning Schedule"                       │
│  [Dashboard] [Tasks] [Import CSV]                  │
├────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────────────────────────────────────┐ │
│  │  Previous Week: Nov 11-17                     │ │
│  │  ┌────────────┐  ┌────────────┐              │ │
│  │  │    Dru     │  │   Josie    │              │ │
│  │  │  3 tasks   │  │  4 tasks   │              │ │
│  │  │  75 mins   │  │  90 mins   │              │ │
│  │  │            │  │            │              │ │
│  │  │ □ Task A   │  │ ☑ Task D   │              │ │
│  │  │ ☑ Task B   │  │ □ Task E   │              │ │
│  │  │ □ Task C   │  │ ☑ Task F   │              │ │
│  │  │            │  │ □ Task G   │              │ │
│  │  └────────────┘  └────────────┘              │ │
│  └──────────────────────────────────────────────┘ │
│                                                     │
│  ┌──────────────────────────────────────────────┐ │
│  │  Current Week: Nov 18-24  ← HIGHLIGHTED      │ │
│  │  [Same layout as above]                       │ │
│  └──────────────────────────────────────────────┘ │
│                                                     │
│  ┌──────────────────────────────────────────────┐ │
│  │  Next Week: Nov 25-Dec 1                      │ │
│  │  [Same layout as above]                       │ │
│  └──────────────────────────────────────────────┘ │
│                                                     │
└────────────────────────────────────────────────────┘
```

**Key Features:**
- Each week is a DaisyUI card
- Current week has border highlight
- Two-column layout within each week
- Checkboxes use HTMX for instant updates
- Task names show category in subtle text
- Completed tasks have strikethrough + opacity

#### 2. Task Management (`/tasks`)

**Layout:**
```
┌────────────────────────────────────────────────────┐
│  Header: "Manage Tasks"                            │
│  [+ Add Task] [Import CSV] [Rebalance]             │
├────────────────────────────────────────────────────┤
│                                                     │
│  Filter: [All] [Bathroom] [Kitchen] [Laundry]      │
│                                                     │
│  ┌──────────────────────────────────────────────┐ │
│  │  Bathroom                                     │ │
│  │  ┌──────────────────────────────────────────┐│ │
│  │  │ General Bathroom                          ││ │
│  │  │ Weekly • 30 mins • Auto-assign            ││ │
│  │  │               [Edit] [Delete]             ││ │
│  │  └──────────────────────────────────────────┘│ │
│  │  ┌──────────────────────────────────────────┐│ │
│  │  │ Clean Bathtub                             ││ │
│  │  │ Monthly • 30 mins • Auto-assign           ││ │
│  │  │               [Edit] [Delete]             ││ │
│  │  └──────────────────────────────────────────┘│ │
│  └──────────────────────────────────────────────┘ │
│                                                     │
│  ┌──────────────────────────────────────────────┐ │
│  │  Kitchen                                      │ │
│  │  [...task cards...]                           │ │
│  └──────────────────────────────────────────────┘ │
│                                                     │
└────────────────────────────────────────────────────┘
```

**Features:**
- Grouped by category
- Click "Add Task" opens modal form
- Edit/Delete use HTMX for instant updates
- Show current assignment (Dru/Josie/Auto)
- Rebalance button recalculates distribution

#### 3. Dashboard (`/dashboard`)

**Layout:**
```
┌────────────────────────────────────────────────────┐
│  Header: "Cleaning Analytics"                      │
│  Period: [Last 7 Days] [Last 30 Days] [Last 90 Days]│
├────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────────┐  ┌──────────────────┐       │
│  │  Completion Rate │  │ Workload Balance │       │
│  │                  │  │                  │       │
│  │  Dru:   87%      │  │  Dru:  245 mins  │       │
│  │  Josie: 92%      │  │  Josie: 250 mins │       │
│  │                  │  │                  │       │
│  │  Overall: 89%    │  │  Diff: 5 mins    │       │
│  └──────────────────┘  └──────────────────┘       │
│                                                     │
│  ┌───────────────────────────────────────────────┐│
│  │  Tasks Completed by Category                  ││
│  │                                                ││
│  │  Bathroom:  ████████░░ 80%                    ││
│  │  Kitchen:   ██████████ 100%                   ││
│  │  Laundry:   ████░░░░░░ 40%                    ││
│  │  Lounge:    ██████░░░░ 60%                    ││
│  └───────────────────────────────────────────────┘│
│                                                     │
│  ┌───────────────────────────────────────────────┐│
│  │  Completion Trend (Last 30 Days)              ││
│  │                                                ││
│  │    [Simple line chart or bar chart]           ││
│  │                                                ││
│  └───────────────────────────────────────────────┘│
│                                                     │
└────────────────────────────────────────────────────┘
```

**Stats to Display:**
- Completion rate per person (%)
- Total time committed vs completed
- Workload balance (total minutes per person)
- Category breakdown
- Streak tracking (optional)
- Overdue tasks count

### Responsive Design

**Mobile (<768px):**
- Stack week columns vertically (Dru above Josie)
- Single column task list
- Hamburger menu for navigation
- Full-width cards

**Tablet (768-1024px):**
- Two-column layout maintained
- Slightly reduced spacing

**Desktop (>1024px):**
- Max width of 1200px, centered
- Optimal spacing for readability

### HTMX Interaction Patterns

#### 1. Checkbox Toggle
```html
<input 
    type="checkbox" 
    hx-post="/tasks/instances/42/toggle"
    hx-swap="outerHTML"
    hx-target="closest .task-row"
/>
```

#### 2. Week Navigation
```html
<button 
    hx-get="/schedule/week/2025-11-25"
    hx-swap="outerHTML"
    hx-target="#next-week-card"
>
    Next Week →
</button>
```

#### 3. Delete Task with Confirmation
```html
<button 
    hx-delete="/tasks/42"
    hx-confirm="Delete this task and all future instances?"
    hx-target="closest .task-card"
    hx-swap="outerHTML swap:1s"
>
    Delete
</button>
```

#### 4. Filter Tasks (Client-side)
```html
<button 
    hx-get="/tasks?category=Kitchen"
    hx-target="#task-list"
    hx-push-url="true"
>
    Kitchen
</button>
```

---

## Business Logic

### 1. Frequency Parsing

**Input Formats:**
- `Daily` → 1 day
- `Weekly` → 7 days
- `Fortnightly` → 14 days
- `Monthly` → 30 days
- `N Weekly` → N * 7 days (e.g., "2 Weekly" = 14 days, "12 Weekly" = 84 days)
- `N x/week` → 7/N days (e.g., "2x/week" = 3.5 days)

**Implementation:**
```go
func ParseFrequency(freq string) (days float64, err error) {
    freq = strings.TrimSpace(strings.ToLower(freq))
    
    switch freq {
    case "daily":
        return 1, nil
    case "weekly":
        return 7, nil
    case "fortnightly":
        return 14, nil
    case "monthly":
        return 30, nil
    default:
        // Handle "N Weekly" pattern
        if strings.HasSuffix(freq, "weekly") {
            parts := strings.Fields(freq)
            if len(parts) == 2 {
                n, err := strconv.Atoi(parts[0])
                if err == nil {
                    return float64(n * 7), nil
                }
            }
        }
        // Handle "N x/week" pattern
        if strings.Contains(freq, "x/week") {
            parts := strings.Split(freq, "x/week")
            if len(parts) == 2 {
                n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
                if err == nil {
                    return 7.0 / float64(n), nil
                }
            }
        }
        return 0, fmt.Errorf("unknown frequency: %s", freq)
    }
}
```

### 2. Fair Task Distribution Algorithm

**Goal:** Distribute unassigned tasks so Dru and Josie have equal total time commitment per week.

**Algorithm (Greedy):**
```
1. Calculate weekly time for each task:
   weekly_time = estimated_mins / (frequency_in_days / 7)
   
2. Separate tasks:
   - assigned_to_dru = tasks with default_assignee='Dru'
   - assigned_to_josie = tasks with default_assignee='Josie'
   - unassigned = tasks with default_assignee=NULL
   
3. Calculate initial totals:
   dru_total = sum(weekly_time for assigned_to_dru)
   josie_total = sum(weekly_time for assigned_to_josie)
   
4. Sort unassigned tasks by weekly_time (descending)

5. For each task in unassigned:
   if dru_total <= josie_total:
       assign to Dru
       dru_total += task.weekly_time
   else:
       assign to Josie
       josie_total += task.weekly_time
       
6. Update tasks table with assignments
```

**Example:**
```
Task A: 30 mins weekly (30/1 = 30 mins/week)
Task B: 60 mins monthly (60/4.3 = 14 mins/week)
Task C: 90 mins fortnightly (90/2 = 45 mins/week)

Sorted by weekly_time: [C(45), A(30), B(14)]

Initial: Dru=0, Josie=0
Assign C to Dru: Dru=45, Josie=0
Assign A to Josie: Dru=45, Josie=30
Assign B to Dru: Dru=59, Josie=30

Result: Dru gets C+B (59), Josie gets A (30)
```

### 3. Task Instance Generation

**Process:**
```
For each task:
    1. Determine first occurrence:
       - Random date between start_date and (start_date + frequency)
       - This adds variety to when tasks first appear
       
    2. Generate recurring instances:
       next_date = first_occurrence
       while next_date <= (today + weeks_ahead * 7):
           create instance:
               task_id = task.id
               scheduled_date = next_date
               assigned_to = task assignment (from distributor)
               week_start_date = Monday of week containing next_date
           next_date += frequency_in_days
```

**Week Start Calculation:**
```go
func GetWeekStart(date time.Time) time.Time {
    // Go's time.Weekday: Sunday=0, Monday=1, ..., Saturday=6
    weekday := int(date.Weekday())
    if weekday == 0 {
        weekday = 7 // Sunday should be 7 for this calc
    }
    daysFromMonday := weekday - 1
    monday := date.AddDate(0, 0, -daysFromMonday)
    return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}
```

**Randomization Example:**
```
Task: "General Bathroom", frequency=Weekly (7 days)
Start date: Nov 16, 2025

Random offset: rand(0-7) = 3 days
First occurrence: Nov 19, 2025

Subsequent:
Nov 19 + 7 = Nov 26
Nov 26 + 7 = Dec 3
Dec 3 + 7 = Dec 10
...
```

### 4. Completion Toggle Logic

```go
func ToggleCompletion(instanceID int, userID string) error {
    // Check if already completed
    completion, err := db.GetCompletion(instanceID)
    
    if err == sql.ErrNoRows {
        // Not completed yet - create completion
        return db.CreateCompletion(instanceID, userID)
    } else if err != nil {
        return err
    } else {
        // Already completed - remove completion (toggle off)
        return db.DeleteCompletion(completion.ID)
    }
}
```

### 5. Analytics Calculations

**Completion Rate:**
```
For period (e.g., last 30 days):
    scheduled = count(instances where scheduled_date in period)
    completed = count(completions joined to instances in period)
    rate = (completed / scheduled) * 100
```

**Workload Balance:**
```
For period:
    dru_scheduled_mins = sum(estimated_mins for Dru's instances)
    josie_scheduled_mins = sum(estimated_mins for Josie's instances)
    difference = abs(dru_scheduled_mins - josie_scheduled_mins)
    balance_percentage = (min / max) * 100
```

**Category Breakdown:**
```
For each category:
    total_in_category = count(instances for category in period)
    completed_in_category = count(completed instances for category)
    category_rate = (completed / total) * 100
```

---

## Implementation Plan

### Phase 1: Project Setup & Database (2-3 hours)

**Step 1.1: Initialize Go Project**
```bash
mkdir cleaning-scheduler
cd cleaning-scheduler
go mod init github.com/yourusername/cleaning-scheduler
mkdir -p cmd/server internal/{database,handlers,scheduler,analytics,models,middleware} web/{templates,static}
```

**Step 1.2: Install Dependencies**
```bash
go get github.com/go-chi/chi/v5
go get github.com/mattn/go-sqlite3
go get github.com/go-co-op/gocron
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
```

**Step 1.3: Create Database Schema**

Create `internal/database/migrations/001_initial_schema.sql`:
```sql
-- +goose Up
CREATE TABLE tasks (...);
CREATE TABLE task_instances (...);
CREATE TABLE completions (...);
CREATE TABLE settings (...);

-- Indexes
CREATE INDEX ...;

-- +goose Down
DROP TABLE completions;
DROP TABLE task_instances;
DROP TABLE tasks;
DROP TABLE settings;
```

**Step 1.4: Setup sqlc**

Create `sqlc.yaml`:
```yaml
version: "2"
sql:
  - schema: "internal/database/migrations"
    queries: "internal/database/queries"
    engine: "sqlite"
    gen:
      go:
        package: "sqlc"
        out: "internal/database/sqlc"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
```

**Step 1.5: Write Database Queries**

Create files in `internal/database/queries/`:
- `tasks.sql` - CRUD for tasks
- `instances.sql` - Instance queries
- `completions.sql` - Completion queries
- `analytics.sql` - Stats queries

Run: `sqlc generate`

**Step 1.6: Database Connection Layer**

Create `internal/database/db.go`:
```go
package database

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "github.com/yourusername/cleaning-scheduler/internal/database/sqlc"
)

func NewDB(dbPath string) (*sqlc.Queries, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }
    return sqlc.New(db), nil
}
```

**Deliverables:**
- ✅ Go project structure
- ✅ Database migrations
- ✅ sqlc queries and generated code
- ✅ Database connection setup

---

### Phase 2: Core Business Logic (3-4 hours)

**Step 2.1: Frequency Parser**

Create `internal/scheduler/frequency.go`:
```go
package scheduler

func ParseFrequency(freq string) (float64, error) {
    // Implementation as described in Business Logic section
}
```

**Step 2.2: Task Distributor**

Create `internal/scheduler/distributor.go`:
```go
package scheduler

type TaskAssignment struct {
    TaskID     int
    AssignedTo string
    WeeklyMins float64
}

func DistributeTasks(tasks []Task) (map[int]string, error) {
    // Implementation of greedy algorithm
}
```

**Step 2.3: Instance Generator**

Create `internal/scheduler/generator.go`:
```go
package scheduler

func GenerateInstances(
    db *sqlc.Queries,
    startDate time.Time,
    weeksAhead int,
) error {
    // For each task:
    //   1. Calculate first occurrence (randomized)
    //   2. Generate recurring instances
    //   3. Insert into task_instances table
}
```

**Step 2.4: Analytics Engine**

Create `internal/analytics/stats.go`:
```go
package analytics

type Stats struct {
    CompletionRateDru    float64
    CompletionRateJosie  float64
    WorkloadDru          int
    WorkloadJosie        int
    CategoryBreakdown    map[string]float64
}

func CalculateStats(db *sqlc.Queries, period int) (*Stats, error) {
    // Implementation of analytics calculations
}
```

**Step 2.5: Unit Tests**

Create test files for each component:
- `frequency_test.go`
- `distributor_test.go`
- `generator_test.go`
- `stats_test.go`

**Deliverables:**
- ✅ Frequency parsing with tests
- ✅ Fair distribution algorithm with tests
- ✅ Instance generation logic with tests
- ✅ Analytics calculations with tests

---

### Phase 3: HTTP Handlers & Routing (4-5 hours)

**Step 3.1: Base Handler Structure**

Create `internal/handlers/helpers.go`:
```go
package handlers

import (
    "html/template"
    "github.com/yourusername/cleaning-scheduler/internal/database/sqlc"
)

type Handler struct {
    db        *sqlc.Queries
    templates *template.Template
}

func NewHandler(db *sqlc.Queries) *Handler {
    tmpl := template.Must(template.ParseGlob("web/templates/**/*.html"))
    return &Handler{db: db, templates: tmpl}
}

func (h *Handler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
    // Helper for rendering templates
}
```

**Step 3.2: Schedule Handlers**

Create `internal/handlers/schedule.go`:
```go
package handlers

func (h *Handler) ScheduleView(w http.ResponseWriter, r *http.Request) {
    // Get current week
    // Query instances for prev/current/next week
    // Fetch completion status
    // Render schedule.html
}

func (h *Handler) GetWeek(w http.ResponseWriter, r *http.Request) {
    // HTMX endpoint for single week
    // Parse date from URL
    // Query instances for that week
    // Render week-card.html partial
}
```

**Step 3.3: Task Handlers**

Create `internal/handlers/tasks.go`:
```go
package handlers

func (h *Handler) TasksList(w http.ResponseWriter, r *http.Request) {
    // Query all tasks grouped by category
    // Render tasks.html
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
    // Parse form data
    // Validate
    // Insert into DB
    // Trigger instance generation
    // Return redirect or HTMX response
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
    // Similar to CreateTask
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
    // Delete task (cascade deletes instances)
    // Return HTMX swap to remove from DOM
}

func (h *Handler) RebalanceTasks(w http.ResponseWriter, r *http.Request) {
    // Run distributor algorithm
    // Update assignments
    // Regenerate instances
    // Redirect to tasks page
}
```

**Step 3.4: Completion Handlers**

Create `internal/handlers/completion.go`:
```go
package handlers

func (h *Handler) ToggleCompletion(w http.ResponseWriter, r *http.Request) {
    instanceID := chi.URLParam(r, "id")
    
    // Toggle completion in DB
    // Query updated instance
    // Render task-row.html partial
}
```

**Step 3.5: Dashboard Handlers**

Create `internal/handlers/dashboard.go`:
```go
package handlers

func (h *Handler) DashboardView(w http.ResponseWriter, r *http.Request) {
    period := r.URL.Query().Get("period") // "7", "30", "90"
    stats := analytics.CalculateStats(h.db, parsePeriod(period))
    h.renderTemplate(w, "dashboard.html", stats)
}

func (h *Handler) RefreshStats(w http.ResponseWriter, r *http.Request) {
    // HTMX endpoint to refresh stats widget
    // Render stats-widget.html partial
}
```

**Step 3.6: Import Handler**

Create `internal/handlers/import.go`:
```go
package handlers

func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form
    // Read CSV file
    // Validate rows
    // Insert tasks into DB
    // Run distributor
    // Generate initial instances
    // Redirect to schedule
}
```

**Step 3.7: Router Setup**

Create `cmd/server/main.go`:
```go
package main

import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    // ... imports
)

func main() {
    // Initialize DB
    db, _ := database.NewDB("cleaning.db")
    
    // Run migrations
    // ...
    
    // Initialize handlers
    h := handlers.NewHandler(db)
    
    // Setup router
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    
    // Static files
    r.Handle("/static/*", http.StripPrefix("/static/", 
        http.FileServer(http.Dir("web/static"))))
    
    // Routes
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, "/schedule", http.StatusSeeOther)
    })
    r.Get("/schedule", h.ScheduleView)
    r.Get("/schedule/week/{date}", h.GetWeek)
    r.Get("/tasks", h.TasksList)
    r.Post("/tasks", h.CreateTask)
    r.Put("/tasks/{id}", h.UpdateTask)
    r.Delete("/tasks/{id}", h.DeleteTask)
    r.Post("/tasks/rebalance", h.RebalanceTasks)
    r.Post("/tasks/instances/{id}/toggle", h.ToggleCompletion)
    r.Get("/dashboard", h.DashboardView)
    r.Post("/tasks/import", h.ImportCSV)
    
    // Start server
    http.ListenAndServe(":8080", r)
}
```

**Deliverables:**
- ✅ All HTTP handlers implemented
- ✅ Router configured with all endpoints
- ✅ Main server entry point ready

---

### Phase 4: HTML Templates (5-6 hours)

**Step 4.1: Base Layout**

Create `web/templates/layout/base.html`:
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
<body class="bg-base-200">
    {{template "nav" .}}
    <main class="container mx-auto px-4 py-8 max-w-6xl">
        {{template "content" .}}
    </main>
</body>
</html>
```

**Step 4.2: Navigation**

Create `web/templates/layout/nav.html`:
```html
{{define "nav"}}
<nav class="navbar bg-base-100 shadow-lg mb-8">
    <div class="flex-1">
        <a href="/" class="btn btn-ghost text-xl">🧹 Cleaning Scheduler</a>
    </div>
    <div class="flex-none">
        <ul class="menu menu-horizontal px-1">
            <li><a href="/schedule">Schedule</a></li>
            <li><a href="/tasks">Tasks</a></li>
            <li><a href="/dashboard">Dashboard</a></li>
        </ul>
    </div>
</nav>
{{end}}
```

**Step 4.3: Schedule Page**

Create `web/templates/pages/schedule.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <h1 class="text-3xl font-bold">Cleaning Schedule</h1>
    
    <!-- Previous Week -->
    <div id="previous-week-card">
        {{template "week-card" .PreviousWeek}}
    </div>
    
    <!-- Current Week (Highlighted) -->
    <div id="current-week-card" class="ring-2 ring-primary">
        {{template "week-card" .CurrentWeek}}
    </div>
    
    <!-- Next Week -->
    <div id="next-week-card">
        {{template "week-card" .NextWeek}}
    </div>
</div>
{{end}}
```

**Step 4.4: Week Card Partial**

Create `web/templates/partials/week-card.html`:
```html
{{define "week-card"}}
<div class="card bg-base-100 shadow-xl">
    <div class="card-body">
        <h2 class="card-title">
            {{.WeekLabel}}
            <span class="text-sm font-normal text-base-content/60">
                {{.StartDate}} - {{.EndDate}}
            </span>
        </h2>
        
        <div class="grid md:grid-cols-2 gap-4 mt-4">
            <!-- Dru's Column -->
            <div>
                <div class="badge badge-primary mb-2">
                    Dru • {{.DruTaskCount}} tasks • {{.DruTotalMins}} mins
                </div>
                <div class="space-y-2">
                    {{range .DruTasks}}
                        {{template "task-row" .}}
                    {{end}}
                </div>
            </div>
            
            <!-- Josie's Column -->
            <div>
                <div class="badge badge-secondary mb-2">
                    Josie • {{.JosieTaskCount}} tasks • {{.JosieTotalMins}} mins
                </div>
                <div class="space-y-2">
                    {{range .JosieTasks}}
                        {{template "task-row" .}}
                    {{end}}
                </div>
            </div>
        </div>
    </div>
</div>
{{end}}
```

**Step 4.5: Task Row Partial**

Create `web/templates/partials/task-row.html`:
```html
{{define "task-row"}}
<div class="task-row" id="task-{{.ID}}">
    <label class="cursor-pointer label">
        <input 
            type="checkbox" 
            {{if .IsCompleted}}checked{{end}}
            class="checkbox {{if eq .AssignedTo "Dru"}}checkbox-primary{{else}}checkbox-secondary{{end}}"
            hx-post="/tasks/instances/{{.ID}}/toggle"
            hx-swap="outerHTML"
            hx-target="closest .task-row"
        />
        <span class="label-text ml-3 {{if .IsCompleted}}line-through opacity-50{{end}}">
            {{.TaskName}}
            <span class="text-xs text-base-content/60">({{.EstimatedMins}} mins)</span>
            <span class="text-xs text-base-content/40">• {{.Category}}</span>
        </span>
    </label>
</div>
{{end}}
```

**Step 4.6: Tasks Page**

Create `web/templates/pages/tasks.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <div class="flex justify-between items-center">
        <h1 class="text-3xl font-bold">Manage Tasks</h1>
        <div class="space-x-2">
            <button class="btn btn-primary" onclick="new_task_modal.showModal()">
                + Add Task
            </button>
            <form method="POST" action="/tasks/rebalance" class="inline">
                <button type="submit" class="btn btn-outline">Rebalance</button>
            </form>
        </div>
    </div>
    
    <!-- Category filters -->
    <div class="flex gap-2 flex-wrap">
        <button class="btn btn-sm" hx-get="/tasks" hx-target="#task-list">All</button>
        {{range .Categories}}
            <button class="btn btn-sm btn-outline" 
                hx-get="/tasks?category={{.}}" 
                hx-target="#task-list">
                {{.}}
            </button>
        {{end}}
    </div>
    
    <!-- Task list -->
    <div id="task-list">
        {{range .TasksByCategory}}
            <div class="mb-6">
                <h3 class="text-xl font-semibold mb-3">{{.Category}}</h3>
                <div class="space-y-2">
                    {{range .Tasks}}
                        {{template "task-card" .}}
                    {{end}}
                </div>
            </div>
        {{end}}
    </div>
</div>

<!-- New Task Modal -->
<dialog id="new_task_modal" class="modal">
    <div class="modal-box">
        <h3 class="font-bold text-lg">Add New Task</h3>
        {{template "task-form" .}}
    </div>
</dialog>
{{end}}
```

**Step 4.7: Task Card Partial**

Create `web/templates/partials/task-card.html`:
```html
{{define "task-card"}}
<div class="card bg-base-100 border border-base-300">
    <div class="card-body py-4 px-6">
        <div class="flex justify-between items-center">
            <div>
                <h4 class="font-semibold">{{.Name}}</h4>
                <p class="text-sm text-base-content/60">
                    {{.Frequency}} • {{.EstimatedMins}} mins • 
                    {{if .DefaultAssignee}}{{.DefaultAssignee}}{{else}}Auto-assign{{end}}
                </p>
            </div>
            <div class="space-x-2">
                <button class="btn btn-sm btn-ghost">Edit</button>
                <button class="btn btn-sm btn-ghost text-error"
                    hx-delete="/tasks/{{.ID}}"
                    hx-confirm="Delete this task?"
                    hx-target="closest .card"
                    hx-swap="outerHTML swap:0.5s">
                    Delete
                </button>
            </div>
        </div>
    </div>
</div>
{{end}}
```

**Step 4.8: Task Form Partial**

Create `web/templates/partials/task-form.html`:
```html
{{define "task-form"}}
<form method="POST" action="/tasks" class="space-y-4">
    <div class="form-control">
        <label class="label"><span class="label-text">Task Name</span></label>
        <input type="text" name="name" class="input input-bordered" required />
    </div>
    
    <div class="form-control">
        <label class="label"><span class="label-text">Category</span></label>
        <input type="text" name="category" class="input input-bordered" required />
    </div>
    
    <div class="form-control">
        <label class="label"><span class="label-text">Frequency</span></label>
        <select name="frequency" class="select select-bordered">
            <option>Daily</option>
            <option selected>Weekly</option>
            <option>Fortnightly</option>
            <option>Monthly</option>
        </select>
    </div>
    
    <div class="form-control">
        <label class="label"><span class="label-text">Estimated Minutes</span></label>
        <input type="number" name="estimated_mins" class="input input-bordered" required />
    </div>
    
    <div class="form-control">
        <label class="label"><span class="label-text">Assign To</span></label>
        <select name="default_assignee" class="select select-bordered">
            <option value="">Auto-assign</option>
            <option value="Dru">Dru</option>
            <option value="Josie">Josie</option>
        </select>
    </div>
    
    <div class="modal-action">
        <button type="submit" class="btn btn-primary">Save Task</button>
        <button type="button" class="btn" onclick="new_task_modal.close()">Cancel</button>
    </div>
</form>
{{end}}
```

**Step 4.9: Dashboard Page**

Create `web/templates/pages/dashboard.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <div class="flex justify-between items-center">
        <h1 class="text-3xl font-bold">Cleaning Analytics</h1>
        <div class="join">
            <button class="join-item btn btn-sm" 
                hx-get="/dashboard?period=7" 
                hx-target="#stats-container">
                7 Days
            </button>
            <button class="join-item btn btn-sm btn-active" 
                hx-get="/dashboard?period=30" 
                hx-target="#stats-container">
                30 Days
            </button>
            <button class="join-item btn btn-sm" 
                hx-get="/dashboard?period=90" 
                hx-target="#stats-container">
                90 Days
            </button>
        </div>
    </div>
    
    <div id="stats-container">
        {{template "stats-widget" .Stats}}
    </div>
</div>
{{end}}
```

**Step 4.10: Stats Widget Partial**

Create `web/templates/partials/stats-widget.html`:
```html
{{define "stats-widget"}}
<div class="grid md:grid-cols-2 gap-6">
    <!-- Completion Rate -->
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
            <h2 class="card-title">Completion Rate</h2>
            <div class="space-y-2">
                <div class="flex justify-between">
                    <span>Dru:</span>
                    <span class="font-semibold">{{.CompletionRateDru}}%</span>
                </div>
                <progress class="progress progress-primary" value="{{.CompletionRateDru}}" max="100"></progress>
                
                <div class="flex justify-between">
                    <span>Josie:</span>
                    <span class="font-semibold">{{.CompletionRateJosie}}%</span>
                </div>
                <progress class="progress progress-secondary" value="{{.CompletionRateJosie}}" max="100"></progress>
                
                <div class="divider"></div>
                
                <div class="flex justify-between font-bold">
                    <span>Overall:</span>
                    <span>{{.OverallRate}}%</span>
                </div>
            </div>
        </div>
    </div>
    
    <!-- Workload Balance -->
    <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
            <h2 class="card-title">Workload Balance</h2>
            <div class="space-y-2">
                <div class="flex justify-between">
                    <span>Dru:</span>
                    <span class="font-semibold">{{.WorkloadDru}} mins</span>
                </div>
                <div class="flex justify-between">
                    <span>Josie:</span>
                    <span class="font-semibold">{{.WorkloadJosie}} mins</span>
                </div>
                <div class="divider"></div>
                <div class="flex justify-between">
                    <span>Difference:</span>
                    <span class="badge {{if lt .WorkloadDiff 30}}badge-success{{else}}badge-warning{{end}}">
                        {{.WorkloadDiff}} mins
                    </span>
                </div>
            </div>
        </div>
    </div>
</div>

<!-- Category Breakdown -->
<div class="card bg-base-100 shadow-xl mt-6">
    <div class="card-body">
        <h2 class="card-title">Completion by Category</h2>
        <div class="space-y-3">
            {{range .CategoryBreakdown}}
                <div>
                    <div class="flex justify-between mb-1">
                        <span>{{.Category}}</span>
                        <span class="font-semibold">{{.Rate}}%</span>
                    </div>
                    <progress class="progress" value="{{.Rate}}" max="100"></progress>
                </div>
            {{end}}
        </div>
    </div>
</div>
{{end}}
```

**Deliverables:**
- ✅ Base layout with navigation
- ✅ Schedule page with 3-week view
- ✅ Task management page with CRUD
- ✅ Dashboard with analytics
- ✅ All HTMX partials for dynamic updates

---

### Phase 5: Background Scheduler (1-2 hours)

**Step 5.1: Cron Setup**

Add to `cmd/server/main.go`:
```go
import (
    "github.com/go-co-op/gocron"
)

func main() {
    // ... existing setup ...
    
    // Initialize background scheduler
    s := gocron.NewScheduler(time.UTC)
    
    // Generate upcoming weeks every Sunday at midnight
    s.Every(1).Week().Sunday().At("00:00").Do(func() {
        log.Println("Running weekly task generation...")
        err := scheduler.GenerateInstances(db, time.Now(), 4)
        if err != nil {
            log.Printf("Error generating instances: %v", err)
        }
    })
    
    // Clean old completions (>90 days) weekly
    s.Every(1).Week().Sunday().At("01:00").Do(func() {
        log.Println("Cleaning old completions...")
        err := db.DeleteOldCompletions(context.Background(), 90)
        if err != nil {
            log.Printf("Error cleaning completions: %v", err)
        }
    })
    
    // Start scheduler
    s.StartAsync()
    
    // ... start HTTP server ...
}
```

**Step 5.2: Initial Generation**

Add startup check in `cmd/server/main.go`:
```go
func main() {
    // ... after DB setup ...
    
    // Check if instances exist, generate if not
    count, err := db.CountInstances(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    
    if count == 0 {
        log.Println("No instances found. Generating initial schedule...")
        startDate := time.Date(2025, 11, 16, 0, 0, 0, 0, time.UTC)
        err = scheduler.GenerateInstances(db, startDate, 4)
        if err != nil {
            log.Fatalf("Failed to generate instances: %v", err)
        }
        log.Println("Initial schedule generated successfully")
    }
    
    // ... continue with cron setup ...
}
```

**Deliverables:**
- ✅ Background scheduler running
- ✅ Weekly instance generation
- ✅ Cleanup of old data
- ✅ Startup initialization

---

### Phase 6: CSV Import (2 hours)

**Step 6.1: CSV Parser**

Update `internal/handlers/import.go`:
```go
package handlers

import (
    "encoding/csv"
    "io"
    "strconv"
)

func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
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
    
    var tasks []TaskInput
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            continue // Skip invalid rows
        }
        
        if len(record) < 5 {
            continue
        }
        
        estimatedMins, _ := strconv.Atoi(record[4])
        
        tasks = append(tasks, TaskInput{
            Name:            record[0],
            Category:        record[1],
            Frequency:       record[2],
            EstimatedMins:   estimatedMins,
            DefaultAssignee: record[3],
        })
    }
    
    // Insert tasks into database
    for _, task := range tasks {
        _, err = h.db.CreateTask(r.Context(), sqlc.CreateTaskParams{
            Name:            task.Name,
            Category:        task.Category,
            Frequency:       task.Frequency,
            EstimatedMins:   int64(task.EstimatedMins),
            DefaultAssignee: sql.NullString{
                String: task.DefaultAssignee,
                Valid:  task.DefaultAssignee != "",
            },
        })
        if err != nil {
            log.Printf("Error creating task %s: %v", task.Name, err)
        }
    }
    
    // Run distribution algorithm
    scheduler.DistributeAndAssign(h.db)
    
    // Generate instances
    startDate := time.Date(2025, 11, 16, 0, 0, 0, 0, time.UTC)
    scheduler.GenerateInstances(h.db, startDate, 4)
    
    // Redirect to schedule
    http.Redirect(w, r, "/schedule", http.StatusSeeOther)
}
```

**Step 6.2: Import UI**

Add to navigation or create import page:
```html
<form method="POST" action="/tasks/import" enctype="multipart/form-data" class="card bg-base-100 shadow-xl">
    <div class="card-body">
        <h2 class="card-title">Import Tasks from CSV</h2>
        <div class="form-control">
            <label class="label">
                <span class="label-text">Upload CSV File</span>
            </label>
            <input type="file" name="csvfile" accept=".csv" class="file-input file-input-bordered" required />
            <label class="label">
                <span class="label-text-alt">Format: Task,Category,Frequency,Assigned To,Estimated Mins</span>
            </label>
        </div>
        <div class="card-actions justify-end">
            <button type="submit" class="btn btn-primary">Import</button>
        </div>
    </div>
</form>
```

**Deliverables:**
- ✅ CSV parsing and validation
- ✅ Bulk task insertion
- ✅ Auto-distribution after import
- ✅ Import UI

---

### Phase 7: Polish & Optimization (2-3 hours)

**Step 7.1: Error Handling**

Add middleware for consistent error pages:
```go
func (h *Handler) ErrorHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic: %v", err)
                h.renderTemplate(w, "error.html", map[string]interface{}{
                    "Error": "Something went wrong",
                })
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

**Step 7.2: Loading States**

Add HTMX loading indicators:
```html
<div class="htmx-indicator">
    <span class="loading loading-spinner loading-lg"></span>
</div>

<style>
.htmx-indicator {
    display: none;
}
.htmx-request .htmx-indicator {
    display: block;
}
</style>
```

**Step 7.3: Validation**

Add form validation:
```go
func ValidateTask(task TaskInput) []string {
    var errors []string
    
    if strings.TrimSpace(task.Name) == "" {
        errors = append(errors, "Task name is required")
    }
    
    if task.EstimatedMins <= 0 {
        errors = append(errors, "Estimated time must be positive")
    }
    
    _, err := scheduler.ParseFrequency(task.Frequency)
    if err != nil {
        errors = append(errors, "Invalid frequency")
    }
    
    return errors
}
```

**Step 7.4: Mobile Optimization**

Add responsive classes:
```html
<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <!-- Auto-stacks on mobile, side-by-side on desktop -->
</div>
```

**Step 7.5: Database Indexes**

Verify indexes in migration:
```sql
CREATE INDEX idx_instances_week ON task_instances(week_start_date);
CREATE INDEX idx_instances_date ON task_instances(scheduled_date);
CREATE INDEX idx_completions_date ON completions(completed_at);
```

**Deliverables:**
- ✅ Error handling middleware
- ✅ Loading indicators
- ✅ Form validation
- ✅ Mobile responsive design
- ✅ Database optimization

---

### Phase 8: Testing (3-4 hours)

**Step 8.1: Unit Tests**

Test coverage for:
- Frequency parsing
- Task distribution algorithm
- Instance generation
- Analytics calculations

**Step 8.2: Integration Tests**

Create `internal/handlers/handlers_test.go`:
```go
func TestScheduleView(t *testing.T) {
    // Setup test DB
    // Create test tasks and instances
    // Make HTTP request
    // Assert response
}

func TestToggleCompletion(t *testing.T) {
    // Test completion toggle
    // Verify DB state changes
}
```

**Step 8.3: Manual Testing Checklist**

- [ ] Import CSV with sample data
- [ ] View schedule shows correct 3 weeks
- [ ] Toggle task completion (on/off)
- [ ] Add new task via UI
- [ ] Edit existing task
- [ ] Delete task (with confirmation)
- [ ] Rebalance workload
- [ ] View dashboard stats
- [ ] Navigate between weeks
- [ ] Mobile responsive layout
- [ ] HTMX updates work without page refresh

**Deliverables:**
- ✅ Unit test suite (>80% coverage)
- ✅ Integration tests for handlers
- ✅ Manual testing completed

---

### Phase 9: Documentation & Deployment (2 hours)

**Step 9.1: README**

Create comprehensive README:
```markdown
# Cleaning Scheduler

A fair and automated household cleaning task scheduler for two people.

## Features
- Automatic fair task distribution
- Weekly schedule view
- Task completion tracking
- Analytics dashboard
- CSV import

## Quick Start

### Prerequisites
- Go 1.21+
- SQLite3

### Installation
```bash
git clone https://github.com/yourusername/cleaning-scheduler
cd cleaning-scheduler
make install
```

### Running
```bash
make run
# Visit http://localhost:8080
```

### Import Sample Data
```bash
# Upload Cleaning.csv via web UI
# Or use sample data: make seed
```

## Development

### Hot Reload
```bash
make dev  # Uses Air for hot reloading
```

### Database Migrations
```bash
make migrate-up
make migrate-down
```

### Testing
```bash
make test
```

## Project Structure
[Include directory tree]

## License
MIT
```

**Step 9.2: Makefile**

Create `Makefile`:
```makefile
.PHONY: build run dev test migrate-up migrate-down install seed

build:
	go build -o bin/cleaning-scheduler cmd/server/main.go

run: build
	./bin/cleaning-scheduler

dev:
	air

test:
	go test -v ./...

migrate-up:
	goose -dir internal/database/migrations sqlite3 cleaning.db up

migrate-down:
	goose -dir internal/database/migrations sqlite3 cleaning.db down

install:
	go mod download
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/cosmtrek/air@latest
	sqlc generate
	make migrate-up

seed:
	sqlite3 cleaning.db < scripts/seed.sql
```

**Step 9.3: Deployment Guide**

Create `DEPLOYMENT.md`:
```markdown
# Deployment Guide

## Local Network Deployment

### Option 1: Run on Server
1. Build: `go build -o cleaning-scheduler cmd/server/main.go`
2. Copy binary and `cleaning.db` to server
3. Run: `./cleaning-scheduler`
4. Access from any device: `http://<server-ip>:8080`

### Option 2: Docker (Optional)
```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o cleaning-scheduler cmd/server/main.go
CMD ["./cleaning-scheduler"]
EXPOSE 8080
```

## Backup
```bash
# Backup database
cp cleaning.db cleaning_backup_$(date +%Y%m%d).db
```

## Environment Variables
- `DB_PATH`: Database file path (default: cleaning.db)
- `PORT`: HTTP server port (default: 8080)
```

**Step 9.4: Air Config**

Create `.air.toml`:
```toml
[build]
  cmd = "go build -o tmp/main cmd/server/main.go"
  bin = "tmp/main"
  include_ext = ["go", "html"]
  exclude_dir = ["tmp", "vendor"]
  delay = 1000

[log]
  time = true
```

**Deliverables:**
- ✅ README with setup instructions
- ✅ Makefile for common tasks
- ✅ Deployment documentation
- ✅ Development tools configured

---

## Testing Strategy

### Unit Tests

**Coverage Targets:**
- Scheduler package: 90%+
- Analytics package: 85%+
- Handlers package: 70%+

**Key Test Cases:**

1. **Frequency Parsing:**
   - Valid inputs (Daily, Weekly, Monthly, etc.)
   - Invalid inputs (error handling)
   - Edge cases ("2 Weekly", "3x/week")

2. **Task Distribution:**
   - Equal distribution with identical tasks
   - Fair distribution with varied time commitments
   - Respect pre-assigned tasks
   - Handle empty task list

3. **Instance Generation:**
   - Correct first occurrence randomization
   - Accurate recurring instances
   - Proper week assignment
   - No duplicate instances

4. **Analytics:**
   - Completion rate calculation
   - Workload balance
   - Category breakdown
   - Handle empty data

### Integration Tests

**Test Scenarios:**

1. **End-to-End Flow:**
   - Import CSV → View schedule → Toggle completion → Check dashboard

2. **HTMX Interactions:**
   - Toggle completion returns correct HTML
   - Week navigation updates view
   - Task deletion removes from DOM

3. **Database Integrity:**
   - Cascading deletes work correctly
   - Unique constraints enforced
   - Foreign keys maintained

### Manual Testing Checklist

**Before Release:**
- [ ] Import provided CSV file successfully
- [ ] All 3 weeks display correctly
- [ ] Task completion persists after refresh
- [ ] New task creation works
- [ ] Task editing preserves data
- [ ] Task deletion removes all instances
- [ ] Rebalance redistributes fairly
- [ ] Dashboard shows accurate stats
- [ ] Mobile layout is usable
- [ ] No console errors in browser
- [ ] Background scheduler runs (check logs after Sunday midnight)

---

## Deployment

### Local Development

```bash
# Clone repository
git clone <repo-url>
cd cleaning-scheduler

# Install dependencies
make install

# Run migrations
make migrate-up

# Start development server with hot reload
make dev

# Visit http://localhost:8080
```

### Production Deployment (Home Server)

**Option 1: Systemd Service**

```bash
# Build binary
make build

# Create systemd service
sudo nano /etc/systemd/system/cleaning-scheduler.service
```

```ini
[Unit]
Description=Cleaning Scheduler
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/home/youruser/cleaning-scheduler
ExecStart=/home/youruser/cleaning-scheduler/bin/cleaning-scheduler
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start
sudo systemctl enable cleaning-scheduler
sudo systemctl start cleaning-scheduler

# Check status
sudo systemctl status cleaning-scheduler
```

**Option 2: Docker**

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o cleaning-scheduler cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /root/
COPY --from=builder /app/cleaning-scheduler .
COPY --from=builder /app/web ./web
EXPOSE 8080
CMD ["./cleaning-scheduler"]
```

```bash
# Build and run
docker build -t cleaning-scheduler .
docker run -d -p 8080:8080 -v $(pwd)/cleaning.db:/root/cleaning.db cleaning-scheduler
```

### Backup Strategy

```bash
# Daily backup script
#!/bin/bash
DATE=$(date +%Y%m%d)
cp cleaning.db backups/cleaning_$DATE.db

# Keep last 30 days
find backups/ -name "cleaning_*.db" -mtime +30 -delete
```

Add to crontab:
```bash
0 2 * * * /home/youruser/cleaning-scheduler/backup.sh
```

---

## Future Enhancements

### Phase 2 Features (Post-MVP)

1. **Task Reassignment**
   - Swap specific instances between people
   - Temporary assignments ("Dru is traveling, Josie handles all")

2. **Enhanced Analytics**
   - Completion streaks
   - Monthly trends chart
   - Overdue task highlighting
   - Time tracking (actual vs estimated)

3. **Notifications**
   - Daily task reminders
   - Overdue alerts
   - Weekly summary emails

4. **Export/Import**
   - Export completion history to CSV
   - Print-friendly schedule view
   - Calendar integration (ICS export)

5. **User Accounts**
   - Basic authentication
   - Multiple households
   - User preferences

6. **Mobile App**
   - React Native or Flutter
   - Push notifications
   - Offline support

### Technical Debt to Address

- Add comprehensive logging
- Implement request rate limiting
- Add database connection pooling
- Create admin panel for settings
- Add data validation middleware
- Improve error messages for users
- Add accessibility (ARIA labels)
- Implement database migrations rollback
- Add health check endpoint
- Create API documentation

---

## Appendix

### A. SQL Queries Reference

**Common Queries:**

```sql
-- Get current week's tasks for a person
SELECT ti.*, t.name, t.estimated_mins, t.category
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
WHERE ti.week_start_date = '2025-11-18'
  AND ti.assigned_to = 'Dru'
ORDER BY ti.scheduled_date;

-- Check completion rate for last 30 days
SELECT 
    assigned_to,
    COUNT(*) as scheduled,
    SUM(CASE WHEN c.id IS NOT NULL THEN 1 ELSE 0 END) as completed,
    ROUND(100.0 * SUM(CASE WHEN c.id IS NOT NULL THEN 1 ELSE 0 END) / COUNT(*), 2) as completion_rate
FROM task_instances ti
LEFT JOIN completions c ON c.task_instance_id = ti.id
WHERE ti.scheduled_date >= date('now', '-30 days')
GROUP BY assigned_to;

-- Get workload balance for current week
SELECT 
    ti.assigned_to,
    SUM(t.estimated_mins) as total_mins
FROM task_instances ti
JOIN tasks t ON ti.task_id = t.id
WHERE ti.week_start_date = date('now', 'weekday 0', '-7 days')
GROUP BY ti.assigned_to;
```

### B. Sample CSV Format

```csv
Task,Category,Frequency,Assigned To,Estimated Mins
General Bathroom,Bathroom,Weekly,,30
Clean Bathtub,Bathroom,Monthly,,30
Mop Bathroom,Bathroom,Monthly,,15
Clean Fans,House,12 Weekly,,60
Empty Fridge,Kitchen,Weekly,,15
Clean Fridge Shelves,Kitchen,12 Weekly,,45
General Kitchen Clean,Kitchen,Weekly,,30
Josie Laundry,Laundry,Weekly,Josie,30
Dru Laundry,Laundry,Weekly,Dru,30
```

### C. Project Dependencies

```go.mod
module github.com/yourusername/cleaning-scheduler

go 1.21

require (
    github.com/go-chi/chi/v5 v5.0.11
    github.com/mattn/go-sqlite3 v1.14.19
    github.com/go-co-op/gocron v1.36.0
)
```

### D. Troubleshooting

**Common Issues:**

1. **Database locked error**
   - Solution: Ensure only one process accesses the DB
   - Use `PRAGMA busy_timeout = 5000;`

2. **HTMX not updating**
   - Check browser console for errors
   - Verify `HX-Request` header in requests
   - Ensure target element exists

3. **Tasks not appearing in schedule**
   - Check if instances were generated: `SELECT COUNT(*) FROM task_instances;`
   - Verify week_start_date calculation
   - Run manual generation: Call `/tasks/regenerate` endpoint

4. **Workload imbalance**
   - Run rebalance manually
   - Check for tasks with incorrect frequency
   - Verify distribution algorithm in logs

---

## Conclusion

This specification provides a complete blueprint for building the Cleaning Scheduler application. A developer following this document should be able to implement the full MVP in approximately 20-25 hours of focused development time.

**Key Success Criteria:**
- Fair task distribution (within 10% time variance)
- Reliable completion tracking (100% persistence)
- Intuitive HTMX-driven UI (no full page reloads)
- Background scheduler runs without intervention
- Mobile-friendly responsive design

**Next Steps:**
1. Set up development environment
2. Follow implementation plan phase by phase
3. Test each phase before moving to next
4. Deploy to local server
5. Import real cleaning tasks CSV
6. Monitor for one week
7. Iterate based on user feedback

For questions or clarifications, refer to the relevant sections above or consult the Go/HTMX/SQLite documentation.
