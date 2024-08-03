-- name: CreateAttachment :one
INSERT INTO attachments (
    todo_id,
    original_filename,
    storage_filename
    ) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetAttachment :one
SELECT * FROM attachments
WHERE id = $1 LIMIT 1;

-- name: ListAttachmentOfTodo :many 
SELECT * FROM attachments
WHERE todo_id = $1 LIMIT 5;

-- name: DeleteAttachment :exec
DELETE FROM attachments
WHERE id = $1;

-- name: DeleteAttachmentsOfTodo :exec
DELETE FROM attachments
WHERE todo_id = $1;
