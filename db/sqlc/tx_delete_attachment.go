package db

import (
	"context"

	storage "github.com/jaingounchained/todo/storage"
)

type DeleteAttachmentTxParams struct {
	TodoID     int64
	Attachment Attachment

	// TODO: Can improve this by returning only relevant closure from Storage instead of whole object
	Storage storage.Storage
}

// DeleteAttachmentTx performs todo information update and file upload
func (store *SQLStore) DeleteAttachmentTx(ctx context.Context, arg DeleteAttachmentTxParams) error {
	return store.execTx(ctx, func(q *Queries) error {
		var err error

		// Decrement file count in todo table
		todo, err := q.UpdateTodoFileCount(ctx, UpdateTodoFileCountParams{
			ID:        arg.TodoID,
			FileCount: int32(-1),
		})
		if err != nil {
			return err
		}

		// Delete attachment record from attachment table
		err = q.DeleteAttachment(ctx, arg.Attachment.ID)
		if err != nil {
			return err
		}

		// Delete file from storage
		return arg.Storage.DeleteFile(ctx, todo.ID, arg.Attachment.StorageFilename)
	})
}
