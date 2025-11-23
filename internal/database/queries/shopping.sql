-- name: CreateShoppingItem :one
INSERT INTO shopping_items (
    name, meal, quantity, week_start_date, added_by
)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetShoppingItem :one
SELECT * FROM shopping_items
WHERE id = ?
LIMIT 1;

-- name: ListShoppingItemsByWeek :many
SELECT * FROM shopping_items
WHERE week_start_date = ?
ORDER BY meal NULLS LAST, name;

-- name: UpdateShoppingItem :one
UPDATE shopping_items
SET name = ?,
    meal = ?,
    quantity = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteShoppingItem :exec
DELETE FROM shopping_items
WHERE id = ?;

-- name: DeleteShoppingItemsByMealAndWeek :exec
DELETE FROM shopping_items
WHERE meal IS NOT NULL
  AND LOWER(meal) = LOWER(sqlc.arg(meal))
  AND week_start_date = sqlc.arg(week_start_date);

-- name: SearchItemsAutocomplete :many
SELECT DISTINCT name, meal, use_count
FROM shopping_item_history
WHERE LOWER(name) LIKE LOWER(?)
ORDER BY use_count DESC, name
LIMIT 10;

-- name: SearchMealsAutocomplete :many
SELECT DISTINCT meal, use_count
FROM shopping_item_history
WHERE meal IS NOT NULL
  AND LOWER(meal) LIKE LOWER(?)
ORDER BY use_count DESC, meal
LIMIT 10;

-- name: GetItemsByMeal :many
SELECT DISTINCT name, meal, quantity
FROM (
    -- Get most recent quantity for each item in this meal
    SELECT name, meal, quantity,
           ROW_NUMBER() OVER (PARTITION BY name ORDER BY created_at DESC) as rn
    FROM shopping_items
    WHERE LOWER(meal) = LOWER(?)
)
WHERE rn = 1
ORDER BY name;

-- name: UpsertItemHistory :exec
INSERT INTO shopping_item_history (name, meal, last_used, use_count)
VALUES (?, ?, CURRENT_TIMESTAMP, 1)
ON CONFLICT(name, meal) DO UPDATE SET
    last_used = CURRENT_TIMESTAMP,
    use_count = use_count + 1;

-- name: GetDistinctMeals :many
SELECT DISTINCT meal
FROM shopping_items
WHERE meal IS NOT NULL
ORDER BY meal;

-- name: DeleteOldShoppingItems :exec
DELETE FROM shopping_items
WHERE week_start_date < date('now', '-90 days');
