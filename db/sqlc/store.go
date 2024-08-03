package db

import (
	"context"
	"database/sql"
)

// Store provides all functions to execute db queries and transaction
type Store interface {
	Querier
	CreateTodoTx(ctx context.Context, arg CreateTodoTxParams) (CreateTodoTxResult, error)
	DeleteTodoTx(ctx context.Context, arg DeleteTodoTxParams) error
	UploadAttachmentTx(ctx context.Context, arg UploadAttachmentTxParams) error
	DeleteAttachmentTx(ctx context.Context, arg DeleteAttachmentTxParams) error
}

// SQLStore provides all functions to execute SQL queries and transaction
type SQLStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		db:      db,
		Queries: New(db),
	}
}
