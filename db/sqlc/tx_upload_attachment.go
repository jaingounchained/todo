package db

import (
	"context"

	storage "github.com/jaingounchained/todo/storage"
	"github.com/jaingounchained/todo/util"
)

// Input parameters for the upload attachment transaction
type UploadAttachmentTxParams struct {
	Todo         Todo
	FileContents storage.FileContents

	// TODO: Can improve this by returning only relevant closure from Storage instead of whole object
	Storage storage.Storage
}

// UploadAttachmentTx performs todo information update and file upload
func (store *SQLStore) UploadAttachmentTx(ctx context.Context, arg UploadAttachmentTxParams) error {
	return store.execTx(ctx, func(q *Queries) error {
		var err error

		// Increment todo file count
		_, err = q.UpdateTodoFileCount(ctx, UpdateTodoFileCountParams{
			ID:        arg.Todo.ID,
			FileCount: int32(len(arg.FileContents)),
		})
		if err != nil {
			return err
		}

		// TODO: Reduce DB calls by inserting attachment metadata in bulk
		// Insert attachment metadata
		StorageFileNameToBytesMap := make(map[string][]byte)
		for fileName := range arg.FileContents {
			uuid, err := util.GenerateUUID()
			if err != nil {
				return err
			}

			_, err = q.CreateAttachment(ctx, CreateAttachmentParams{
				TodoID:           arg.Todo.ID,
				OriginalFilename: fileName,
				StorageFilename:  uuid,
			})
			if err != nil {
				return err
			}

			StorageFileNameToBytesMap[uuid] = arg.FileContents[fileName]
		}

		// Save file
		return arg.Storage.SaveMultipleFilesSafely(ctx, arg.Todo.ID, StorageFileNameToBytesMap)
	})
}
