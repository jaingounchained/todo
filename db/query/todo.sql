-- name: CreateTodo :one
INSERT INTO todos (
    owner,
    title,
    periodic_reminder_time_seconds
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetTodo :one
SELECT * FROM todos
WHERE id = $1 LIMIT 1;

-- name: ListTodos :many
SELECT * FROM todos
WHERE owner = $1
ORDER BY id
LIMIT $2
OFFSET $3;

-- name: UpdateTodo :one
UPDATE todos
SET title = COALESCE(sqlc.narg(title), title),
    status = COALESCE(sqlc.narg(status), status),
    periodic_reminder_time_seconds = COALESCE(sqlc.narg(periodic_reminder_time_seconds), periodic_reminder_time_seconds)
WHERE id = $1
RETURNING *;

-- name: UpdateTodoFileCount :one
UPDATE todos
SET file_count = file_count + $2
WHERE id = $1
RETURNING *;

-- name: DeleteTodo :exec
DELETE FROM todos
WHERE id = $1;
