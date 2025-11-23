-- Meals table for managing unique meal names
CREATE TABLE IF NOT EXISTS meals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create index for meal name lookups
CREATE INDEX IF NOT EXISTS idx_meals_name ON meals(name);

-- Migrate existing meals from shopping_item_history to meals table
INSERT OR IGNORE INTO meals (name)
SELECT DISTINCT meal
FROM shopping_item_history
WHERE meal IS NOT NULL AND meal != '';
