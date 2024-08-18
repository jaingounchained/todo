package db

import (
	"context"
)

// Input parameters for the upload attachment transaction
type CreateTodoTxParams struct {
	CreateTodoParams

	SetupTodoAttachmentStorage func(todo Todo) error
	StartSendingReminders      func(todo Todo) error
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
		result.Todo, err = q.CreateTodo(ctx, arg.CreateTodoParams)
		if err != nil {
			return err
		}

		// Setup storage
		err = arg.SetupTodoAttachmentStorage(result.Todo)
		if err != nil {
			// TODO: Think about how to rollback the storage
			return err
		}

		// Start reminder
		err = arg.StartSendingReminders(result.Todo)
		if err != nil {
			// TODO: Think about how to rollback reminder process if initiated
			return err
		}

		return nil
	})

	return result, err
}
