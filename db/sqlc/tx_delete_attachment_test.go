package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	mockStorage "github.com/jaingounchained/todo/storage/mock"
	"github.com/stretchr/testify/require"
)

// TODO: Improve tests by using anonymous struct
func TestDeleteAttachmentTxOK(t *testing.T) {
	// Setup
	// Insert a todo in DB
	todo := createRandomTodo(t)
	// Increment file count by 2
	_, err := testStore.UpdateTodoFileCount(context.Background(), UpdateTodoFileCountParams{
		ID:        todo.ID,
		FileCount: 2,
	})
	// Insert 2 attachments in DB
	attachment1 := createRandomAttachmentForTodo(t, todo)
	createRandomAttachmentForTodo(t, todo)
	// Increment todo file count in memory
	todo.FileCount += 2

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testMockStorage := mockStorage.NewMockStorage(ctrl)
	testMockStorage.EXPECT().
		DeleteFile(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachment1.StorageFilename)).
		Return(nil).
		Times(1)

	err = testStore.DeleteAttachmentTx(context.Background(), DeleteAttachmentTxParams{
		TodoID:     todo.ID,
		Attachment: attachment1,
		Storage:    testMockStorage,
	})
	// Decrement todo filecount in memory
	todo.FileCount--
	require.NoError(t, err)

	// Query the db to find the todo
	updatedTodo, err := testStore.GetTodo(context.Background(), todo.ID)
	require.NoError(t, err)
	compareTodos(t, updatedTodo, todo)

	// Query the db to find the attachment
	actualAttachment, err := testStore.GetAttachment(context.Background(), attachment1.ID)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, actualAttachment)
}

func TestDeleteAttachmentTxStorageFailure(t *testing.T) {
	// Setup
	// Insert a todo in DB
	todo := createRandomTodo(t)
	// Increment file count by 2
	_, err := testStore.UpdateTodoFileCount(context.Background(), UpdateTodoFileCountParams{
		ID:        todo.ID,
		FileCount: 2,
	})
	// Insert 2 attachments in DB
	attachment1 := createRandomAttachmentForTodo(t, todo)
	createRandomAttachmentForTodo(t, todo)
	// Increment todo file count in memory
	todo.FileCount += 2

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storageError := errors.New("storage error")
	testMockStorage := mockStorage.NewMockStorage(ctrl)
	testMockStorage.EXPECT().
		DeleteFile(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachment1.StorageFilename)).
		Return(storageError).
		Times(1)

	err = testStore.DeleteAttachmentTx(context.Background(), DeleteAttachmentTxParams{
		TodoID:     todo.ID,
		Attachment: attachment1,
		Storage:    testMockStorage,
	})
	require.EqualError(t, err, storageError.Error())

	// Query the db to find the todo
	updatedTodo, err := testStore.GetTodo(context.Background(), todo.ID)
	require.NoError(t, err)
	compareTodos(t, updatedTodo, todo)

	// Query the db to find the attachment
	actualAttachment, err := testStore.GetAttachment(context.Background(), attachment1.ID)
	require.NoError(t, err)
	compareAttachment(t, actualAttachment, attachment1)
}
