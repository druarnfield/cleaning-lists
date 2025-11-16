# Cleaning Scheduler - Template Issues Diagnosis and Fixes

## Issues Identified

### 1. **Go Template Pattern Incorrect**
**Problem**: All page templates use this anti-pattern:
```html
{{define "pagename.html"}}
{{template "base.html" .}}
{{end}}

{{define "content"}}
...actual content...
{{end}}
```

**Why this fails**: When `{{template "base.html" .}}` executes, it looks for the `content` block, but that block is defined AFTER the template call. Go templates parse sequentially, so the `content` block doesn't exist yet when `base.html` tries to use it.

**Files affected**:
- web/templates/auth/setup_password.html
- web/templates/auth/reset_password.html
- web/templates/pages/schedule.html
- web/templates/pages/dashboard.html
- web/templates/pages/tasks.html
- web/templates/pages/import.html

### 2. **Template Parsing Incomplete**
**Problem**: In `internal/handlers/base.go`:
```go
tmpl := template.Must(template.ParseGlob("web/templates/*/*.html"))
```

This glob pattern only goes ONE level deep (e.g., `web/templates/auth/*.html`) and doesn't recursively parse all subdirectories.

### 3. **Inconsistent Template Execution**
**Problem**: The `render()` function calls:
```go
h.templates.ExecuteTemplate(w, name, data)
```

But the template names in the code don't match the `{{define}}` blocks in the templates. For example, calling `render(w, "login.html", data)` but the template is defined as `{{define "login.html"}}`.

### 4. **Missing User Context in Navigation**
**Problem**: `web/templates/layout/nav.html` expects `.User` in the data context, but not all handlers provide it. This causes the navigation to not render properly on protected pages.

## Solution: Complete Template Restructure

### Step 1: Fix the Base Template Pattern

**Option A: Recommended - Wrapper Template Approach**

Replace `web/templates/layout/base.html` with:

```html
{{define "base.html"}}
<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{if .Title}}{{.Title}}{{else}}Cleaning Scheduler{{end}}</title>
    <link href="https://cdn.jsdelivr.net/npm/daisyui@4.4.0/dist/full.min.css" rel="stylesheet">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-base-200 min-h-screen">
    {{if .User}}
        {{template "nav.html" .}}
    {{end}}
    <main class="container mx-auto px-4 py-8 max-w-6xl">
        {{block "content" .}}{{end}}
    </main>
</body>
</html>
{{end}}
```

Key changes:
- Use `{{block "content" .}}{{end}}` instead of `{{template "content" .}}`
- Added conditional navigation (only show if `.User` exists)
- Made title optional

### Step 2: Fix All Page Templates

Each page template should follow this pattern:

**Example: `web/templates/auth/setup_password.html`**
```html
{{define "setup_password.html"}}
{{template "base.html" .}}

{{define "content"}}
<div class="flex justify-center items-center min-h-[60vh]">
    <div class="card w-96 bg-base-100 shadow-xl">
        <div class="card-body">
            <h2 class="card-title text-2xl mb-4">Set Your Password</h2>
            <p class="text-base-content/60 mb-4">Welcome {{.User.Username}}! Please set your password.</p>

            {{if .Error}}
            <div class="alert alert-error mb-4">
                <span>{{.Error}}</span>
            </div>
            {{end}}

            <form method="POST" action="/setup-password">
                <div class="form-control mb-4">
                    <label class="label">
                        <span class="label-text">New Password</span>
                    </label>
                    <input type="password" name="new_password" class="input input-bordered" required minlength="6">
                </div>

                <div class="form-control mb-6">
                    <label class="label">
                        <span class="label-text">Confirm Password</span>
                    </label>
                    <input type="password" name="confirm_password" class="input input-bordered" required minlength="6">
                </div>

                <button type="submit" class="btn btn-primary w-full">Set Password</button>
            </form>
        </div>
    </div>
</div>
{{end}}
{{end}}
```

**Critical pattern**:
```
{{define "templatename.html"}}
{{template "base.html" .}}

{{define "content"}}
...your content here...
{{end}}
{{end}}  <!-- closes the outer define -->
```

Apply this pattern to ALL these files:
- web/templates/auth/reset_password.html
- web/templates/pages/schedule.html
- web/templates/pages/dashboard.html
- web/templates/pages/tasks.html
- web/templates/pages/import.html

### Step 3: Fix Navigation Template

**Update `web/templates/layout/nav.html`:**

```html
{{define "nav.html"}}
<div class="navbar bg-base-100 shadow-lg">
    <div class="flex-1">
        <a href="/" class="btn btn-ghost text-xl">Cleaning Scheduler</a>
    </div>
    <div class="flex-none">
        {{if .User}}
        <ul class="menu menu-horizontal px-1">
            <li><a href="/schedule">Schedule</a></li>
            <li><a href="/tasks">Tasks</a></li>
            <li><a href="/dashboard">Dashboard</a></li>
            <li><a href="/import">Import</a></li>
            <li>
                <details>
                    <summary>{{.User.Username}}</summary>
                    <ul class="p-2 bg-base-100 z-10">
                        <li><a href="/reset-password">Reset Password</a></li>
                        <li>
                            <form method="POST" action="/logout">
                                <button type="submit" class="btn btn-ghost btn-sm w-full text-left">Logout</button>
                            </form>
                        </li>
                    </ul>
                </details>
            </li>
        </ul>
        {{end}}
    </div>
</div>
{{end}}
```

### Step 4: Fix Template Parsing in Handler

**Update `internal/handlers/base.go`:**

```go
package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
	"github.com/druarnfield/cleaning-scheduler/internal/auth"
	"github.com/druarnfield/cleaning-scheduler/internal/database/sqlc"
)

type Handler struct {
	db        *sqlc.Queries
	auth      *auth.AuthService
	templates *template.Template
}

func NewHandler(db *sqlc.Queries, authService *auth.AuthService) *Handler {
	// Parse all templates recursively
	tmpl := template.New("")
	
	// Parse layout templates
	tmpl = template.Must(tmpl.ParseGlob("web/templates/layout/*.html"))
	
	// Parse auth templates
	tmpl = template.Must(tmpl.ParseGlob("web/templates/auth/*.html"))
	
	// Parse page templates
	tmpl = template.Must(tmpl.ParseGlob("web/templates/pages/*.html"))
	
	// Parse partial templates
	tmpl = template.Must(tmpl.ParseGlob("web/templates/partials/*.html"))

	return &Handler{
		db:        db,
		auth:      authService,
		templates: tmpl,
	}
}

func (h *Handler) render(w http.ResponseWriter, name string, data interface{}) {
	// Ensure data is a map so we can add defaults
	dataMap, ok := data.(map[string]interface{})
	if !ok && data != nil {
		// If data is not a map, wrap it
		dataMap = map[string]interface{}{"Data": data}
	} else if data == nil {
		dataMap = make(map[string]interface{})
	}
	
	// Set default title if not provided
	if _, hasTitle := dataMap["Title"]; !hasTitle {
		dataMap["Title"] = "Cleaning Scheduler"
	}
	
	err := h.templates.ExecuteTemplate(w, name, dataMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
```

### Step 5: Ensure User Context is Passed

**Update handler methods to always include User in context:**

For example, in `internal/handlers/schedule.go`:

```go
func (h *Handler) ScheduleView(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())

	// ... your existing logic to get week data ...

	data := map[string]interface{}{
		"User":        user,  // Make sure this is always included
		"Title":       "Schedule",
		"PrevWeek":    formatWeekData(prevWeek, prevWeekStart),
		"CurrentWeek": formatWeekData(currentWeek, currentWeekStart),
		"NextWeek":    formatWeekData(nextWeek, nextWeekStart),
	}

	h.render(w, "schedule.html", data)
}
```

Apply this pattern to ALL protected route handlers:
- ScheduleView
- TasksList
- DashboardView
- ImportCSV (GET)
- ResetPasswordPage

## Quick Fix Commands

Run these commands to fix the templates:

### 1. Backup current templates
```bash
cp -r web/templates web/templates.backup
```

### 2. Fix base.html
```bash
cat > web/templates/layout/base.html << 'EOF'
{{define "base.html"}}
<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{if .Title}}{{.Title}}{{else}}Cleaning Scheduler{{end}}</title>
    <link href="https://cdn.jsdelivr.net/npm/daisyui@4.4.0/dist/full.min.css" rel="stylesheet">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-base-200 min-h-screen">
    {{if .User}}
        {{template "nav.html" .}}
    {{end}}
    <main class="container mx-auto px-4 py-8 max-w-6xl">
        {{block "content" .}}{{end}}
    </main>
</body>
</html>
{{end}}
EOF
```

### 3. Fix each page template
For each template in auth/ and pages/, wrap the entire file like this:

**Before:**
```html
{{define "templatename.html"}}
{{template "base.html" .}}
{{end}}

{{define "content"}}
...content...
{{end}}
```

**After:**
```html
{{define "templatename.html"}}
{{template "base.html" .}}

{{define "content"}}
...content...
{{end}}
{{end}}  <!-- Add this closing tag -->
```

## Testing Steps

After applying all fixes:

1. **Rebuild and run:**
   ```bash
   make build
   make run
   ```

2. **Test login flow:**
   - Navigate to http://localhost:8080
   - Should redirect to /login
   - Select "dru" or "josie"
   - Leave password blank (or enter anything since no password is set)
   - Click Login
   - **Should redirect to /setup-password** ✓

3. **Test password setup:**
   - Enter a new password (min 6 characters)
   - Confirm password
   - Click "Set Password"
   - **Should redirect to /schedule** ✓

4. **Test navigation:**
   - Navigation bar should appear at top
   - Click "Schedule" - should show schedule page
   - Click "Tasks" - should show tasks page
   - Click "Dashboard" - should show dashboard
   - Click "Import" - should show import page
   - All navigation should work ✓

5. **Test logout:**
   - Click your username dropdown
   - Click "Logout"
   - Should redirect to /login ✓

## Summary of Changes

1. ✅ Fixed base.html to use `{{block "content" .}}` instead of `{{template "content" .}}`
2. ✅ Added conditional navigation rendering based on `.User` presence
3. ✅ Fixed all page templates to properly nest define blocks
4. ✅ Updated template parsing to use multiple ParseGlob calls for each directory
5. ✅ Enhanced render function to ensure data is always a map with defaults
6. ✅ Ensured all protected handlers pass User context

## Additional Recommendations

1. **Add template helper functions** for common formatting tasks
2. **Add CSRF protection** to forms
3. **Add client-side validation** for password confirmation
4. **Add loading states** for HTMX requests
5. **Add error pages** (404, 500) using the same template pattern

## File Checklist

Apply the template fix pattern to these files:

- [ ] web/templates/layout/base.html
- [ ] web/templates/layout/nav.html  
- [ ] web/templates/auth/setup_password.html
- [ ] web/templates/auth/reset_password.html
- [ ] web/templates/pages/schedule.html
- [ ] web/templates/pages/dashboard.html
- [ ] web/templates/pages/tasks.html
- [ ] web/templates/pages/import.html
- [ ] internal/handlers/base.go

The partials (task_row.html and week_card.html) are correctly structured and don't need changes.
