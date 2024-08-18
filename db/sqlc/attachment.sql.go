// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: attachment.sql

package db

import (
	"context"
)

const createAttachment = `-- name: CreateAttachment :one
INSERT INTO attachments (
    todo_id,
    original_filename,
    storage_filename
) VALUES (
    $1, $2, $3
) RETURNING id, todo_id, original_filename, storage_filename, created_at
`

type CreateAttachmentParams struct {
	TodoID           int64  `json:"todoId"`
	OriginalFilename string `json:"originalFilename"`
	StorageFilename  string `json:"storageFilename"`
}

func (q *Queries) CreateAttachment(ctx context.Context, arg CreateAttachmentParams) (Attachment, error) {
	row := q.db.QueryRow(ctx, createAttachment, arg.TodoID, arg.OriginalFilename, arg.StorageFilename)
	var i Attachment
	err := row.Scan(
		&i.ID,
		&i.TodoID,
		&i.OriginalFilename,
		&i.StorageFilename,
		&i.CreatedAt,
	)
	return i, err
}

const deleteAttachment = `-- name: DeleteAttachment :exec
DELETE FROM attachments
WHERE id = $1
`

func (q *Queries) DeleteAttachment(ctx context.Context, id int64) error {
	_, err := q.db.Exec(ctx, deleteAttachment, id)
	return err
}

const deleteAttachmentsOfTodo = `-- name: DeleteAttachmentsOfTodo :exec
DELETE FROM attachments
WHERE todo_id = $1
`

func (q *Queries) DeleteAttachmentsOfTodo(ctx context.Context, todoID int64) error {
	_, err := q.db.Exec(ctx, deleteAttachmentsOfTodo, todoID)
	return err
}

const getAttachment = `-- name: GetAttachment :one
SELECT id, todo_id, original_filename, storage_filename, created_at FROM attachments
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetAttachment(ctx context.Context, id int64) (Attachment, error) {
	row := q.db.QueryRow(ctx, getAttachment, id)
	var i Attachment
	err := row.Scan(
		&i.ID,
		&i.TodoID,
		&i.OriginalFilename,
		&i.StorageFilename,
		&i.CreatedAt,
	)
	return i, err
}

const listAttachmentOfTodo = `-- name: ListAttachmentOfTodo :many
SELECT id, todo_id, original_filename, storage_filename, created_at FROM attachments
WHERE todo_id = $1 LIMIT 5
`

func (q *Queries) ListAttachmentOfTodo(ctx context.Context, todoID int64) ([]Attachment, error) {
	rows, err := q.db.Query(ctx, listAttachmentOfTodo, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Attachment{}
	for rows.Next() {
		var i Attachment
		if err := rows.Scan(
			&i.ID,
			&i.TodoID,
			&i.OriginalFilename,
			&i.StorageFilename,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
