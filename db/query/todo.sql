-- name: CreateTodo :one
INSERT INTO todos (
    title
) VALUES (
    $1
) RETURNING *;

-- name: GetTodo :one
SELECT * FROM todos
WHERE id = $1 LIMIT 1;

-- name: ListTodos :many
SELECT * FROM todos
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateTodoTitleStatus :one
UPDATE todos
SET title = COALESCE(sqlc.narg(title), title),
    status = COALESCE(sqlc.narg(status), status)
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
