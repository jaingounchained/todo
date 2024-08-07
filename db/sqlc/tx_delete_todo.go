package db

import (
	"context"
	storage "github.com/jaingounchained/todo/storage"
)

// Input parameters for the upload attachment transaction
type DeleteTodoTxParams struct {
	TodoID int64

	// TODO: Can improve this by returning only relevant closure from Storage instead of whole object
	Storage storage.Storage
}

// DeleteAttachmentTx performs todo information update and file upload
func (store *SQLStore) DeleteTodoTx(ctx context.Context, arg DeleteTodoTxParams) error {
	return store.execTx(ctx, func(q *Queries) error {
		var err error

		// Delete todo and corresponding attachment rows if present
		err = q.DeleteTodo(ctx, arg.TodoID)
		if err != nil {
			return err
		}

		// Delete file
		return arg.Storage.DeleteTodoDirectory(ctx, arg.TodoID)
	})
}
