-- Users table (only Dru and Josie)
CREATE TABLE users (
    username TEXT PRIMARY KEY CHECK(username IN ('dru', 'josie')),
    password_hash TEXT,
    is_password_set BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table for token management
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (username) REFERENCES users(username)
);

-- Core task definitions (long-lived, user editable)
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    frequency TEXT NOT NULL,  -- 'Daily', 'Weekly', 'Fortnightly', 'Monthly', '2 Weekly', etc.
    estimated_mins INTEGER NOT NULL,
    default_assignee TEXT,    -- 'Dru', 'Josie', 'Both', 'Alternate', or NULL for auto-assign
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Scheduled task instances with bring-forward support
CREATE TABLE task_instances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    scheduled_date DATE NOT NULL,
    original_scheduled_date DATE,  -- NULL unless brought forward
    assigned_to TEXT NOT NULL CHECK(assigned_to IN ('dru', 'josie')),
    week_start_date DATE NOT NULL,  -- Monday of the week this task falls in
    brought_forward BOOLEAN DEFAULT FALSE,
    brought_forward_by TEXT,  -- Who brought it forward
    counts_toward_weekly BOOLEAN DEFAULT TRUE,  -- FALSE for brought-forward tasks
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Task completions (tracks when tasks were marked done)
CREATE TABLE completions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_instance_id INTEGER NOT NULL,
    completed_by TEXT NOT NULL CHECK(completed_by IN ('dru', 'josie')),
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
CREATE INDEX idx_instances_brought_forward ON task_instances(brought_forward);
CREATE INDEX idx_completions_instance ON completions(task_instance_id);
CREATE INDEX idx_completions_date ON completions(completed_at);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
CREATE INDEX idx_sessions_username ON sessions(username);

-- Insert default users (passwords not set)
INSERT INTO users (username, is_password_set) VALUES
    ('dru', FALSE),
    ('josie', FALSE);

-- Insert default settings
INSERT INTO settings (key, value) VALUES
    ('start_date', date('now')),
    ('weeks_ahead', '4'),
    ('app_version', '1.0.0'),
    ('session_expiry_days', '30');

