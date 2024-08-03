package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides all functions to execute db queries and transaction
type Store interface {
	Querier
	UploadAttachmentTx(ctx context.Context, arg UploadAttachmentTxParams) (UploadAttachmentTxResult, error)
	DeleteAttachmentTx(ctx context.Context, arg DeleteAttachmentTxParams) (DeleteAttachmentTxResult, error)
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

func (store *SQLStore) execTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// Input parameters for the upload attachment transaction
type UploadAttachmentTxParams struct {
}

// Result of upload attachment transaction
type UploadAttachmentTxResult struct {
}

// UploadAttachmentTx performs todo information update and file upload
func (store *SQLStore) UploadAttachmentTx(ctx context.Context, arg UploadAttachmentTxParams) (UploadAttachmentTxResult, error) {
	var result UploadAttachmentTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		// Increment file number in todo table

		// Add attachment metadata

		// Upload file

		return nil
	})

	return result, err
}

// Input parameters for the upload attachment transaction
type DeleteAttachmentTxParams struct {
}

// Result of upload attachment transaction
type DeleteAttachmentTxResult struct {
}

// DeleteAttachmentTx performs todo information update and file upload
func (store *SQLStore) DeleteAttachmentTx(ctx context.Context, arg DeleteAttachmentTxParams) (DeleteAttachmentTxResult, error) {
	var result DeleteAttachmentTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		// Decrement file number in todo table

		// Delete attachment metadata

		// Delete file

		return nil
	})

	return result, err
}
