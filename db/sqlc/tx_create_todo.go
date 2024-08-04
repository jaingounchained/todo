package db

import (
	"context"

	storage "github.com/jaingounchained/todo/storage"
)

// Input parameters for the upload attachment transaction
type CreateTodoTxParams struct {
	TodoTitle string

	// TODO: Can improve this by returning only relevant closure from Storage instead of whole object
	Storage storage.Storage
}

// Result of upload attachment transaction
type CreateTodoTxResult struct {
	Todo Todo
}

// CreateAttachmentTx performs todo information update and file upload
func (store *SQLStore) CreateTodoTx(ctx context.Context, arg CreateTodoTxParams) (CreateTodoTxResult, error) {
	var result CreateTodoTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		// Insert todo
		result.Todo, err = q.CreateTodo(ctx, arg.TodoTitle)
		if err != nil {
			return err
		}

		// Create file
		return arg.Storage.CreateTodoDirectory(ctx, result.Todo.ID)
	})

	return result, err
}
