-- Add chat history table for schedule suggestions
CREATE TABLE schedule_suggestions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_message TEXT NOT NULL,
  llm_response TEXT NOT NULL,
  week_offset INTEGER,
  changes_made BOOLEAN DEFAULT FALSE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_suggestions_created ON schedule_suggestions(created_at DESC);

-- Delete future task instances (current week and beyond)
-- Historical data (past weeks) is preserved
-- This allows the LLM to generate fresh schedules
DELETE FROM task_instances
WHERE week_start_date >= (
  SELECT date('now', 'weekday 1', '-7 days')
);
