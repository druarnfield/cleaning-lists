-- Shopping list items (week-based shopping lists)
CREATE TABLE IF NOT EXISTS shopping_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    meal TEXT,  -- Meal tag/category (e.g., "Monday Dinner", "Pasta Night")
    quantity INTEGER NOT NULL DEFAULT 1,
    week_start_date DATE NOT NULL,  -- Monday of the week this item is for
    added_by TEXT NOT NULL CHECK(added_by IN ('dru', 'josie')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Shopping item history (for autocomplete)
-- Tracks unique item+meal combinations
CREATE TABLE IF NOT EXISTS shopping_item_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    meal TEXT,  -- Can be NULL for items without meal tags
    last_used DATETIME DEFAULT CURRENT_TIMESTAMP,
    use_count INTEGER DEFAULT 1,
    UNIQUE(name, meal)  -- Ensure unique item+meal combinations
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_shopping_items_week ON shopping_items(week_start_date);
CREATE INDEX IF NOT EXISTS idx_shopping_items_added_by ON shopping_items(added_by);
CREATE INDEX IF NOT EXISTS idx_shopping_items_meal ON shopping_items(meal);
CREATE INDEX IF NOT EXISTS idx_shopping_history_name ON shopping_item_history(name);
CREATE INDEX IF NOT EXISTS idx_shopping_history_meal ON shopping_item_history(meal);
CREATE INDEX IF NOT EXISTS idx_shopping_history_use_count ON shopping_item_history(use_count DESC);
