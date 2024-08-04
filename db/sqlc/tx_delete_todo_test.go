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
func TestDeleteTodoTxOK(t *testing.T) {
	// Setup: Insert a todo in DB
	todo := createRandomTodo(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testMockStorage := mockStorage.NewMockStorage(ctrl)
	var capturedTodoID int64
	testMockStorage.EXPECT().
		DeleteTodoDirectory(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, todoID int64) {
			capturedTodoID = todoID
		}).
		Times(1)

	err := testStore.DeleteTodoTx(context.Background(), DeleteTodoTxParams{
		TodoID:  todo.ID,
		Storage: testMockStorage,
	})
	require.NoError(t, err)

	// Query the db to find the row
	actualTodo, err := testStore.GetTodo(context.Background(), todo.ID)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, actualTodo)

	// Check todoID called in storage
	require.Equal(t, todo.ID, capturedTodoID)
}

func TestDeleteTodoTxStorageFailure(t *testing.T) {
	// Setup: Insert a todo in DB
	todo := createRandomTodo(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testMockStorage := mockStorage.NewMockStorage(ctrl)
	expectedError := errors.New("storage failure")
	var capturedTodoID int64
	testMockStorage.EXPECT().
		DeleteTodoDirectory(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, todoID int64) error {
			capturedTodoID = todoID
			return expectedError
		}).
		Times(1)

	err := testStore.DeleteTodoTx(context.Background(), DeleteTodoTxParams{
		TodoID:  todo.ID,
		Storage: testMockStorage,
	})
	require.Error(t, err)
	require.EqualError(t, err, expectedError.Error())

	// Query the db to find the row
	actualTodo, err := testStore.GetTodo(context.Background(), todo.ID)
	require.NoError(t, err)
	require.Equal(t, actualTodo, todo)

	// Check todoID called in storage
	require.Equal(t, capturedTodoID, todo.ID)
}
