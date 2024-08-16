package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store provides all functions to execute db queries and transaction
type Store interface {
	Querier
	CreateTodoTx(ctx context.Context, arg CreateTodoTxParams) (CreateTodoTxResult, error)
	DeleteTodoTx(ctx context.Context, arg DeleteTodoTxParams) error
	UploadAttachmentTx(ctx context.Context, arg UploadAttachmentTxParams) error
	DeleteAttachmentTx(ctx context.Context, arg DeleteAttachmentTxParams) error
	CreateUserTx(ctx context.Context, arg CreateUserTxParams) (CreateUserTxResult, error)
	VerifyEmailTx(ctx context.Context, arg VerifyEmailTxParams) (VerifyEmailTxResult, error)
}

// SQLStore provides all functions to execute SQL queries and transaction
type SQLStore struct {
	*Queries
	connPool *pgxpool.Pool
}

func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}
