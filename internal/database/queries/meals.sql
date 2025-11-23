-- name: CreateMeal :one
INSERT INTO meals (name)
VALUES (?)
RETURNING *;

-- name: GetMeal :one
SELECT * FROM meals
WHERE id = ?
LIMIT 1;

-- name: GetMealByName :one
SELECT * FROM meals
WHERE LOWER(name) = LOWER(?)
LIMIT 1;

-- name: ListMeals :many
SELECT * FROM meals
ORDER BY name;

-- name: UpdateMeal :one
UPDATE meals
SET name = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: DeleteMeal :exec
DELETE FROM meals
WHERE id = ?;

-- name: CountMealUsage :one
SELECT COUNT(*) FROM shopping_items
WHERE meal IS NOT NULL AND LOWER(meal) = LOWER(?);
