package db

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	mockStorage "github.com/jaingounchained/todo/storage/mock"
	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

// TODO: Improve tests by using anonymous struct
func TestCreateTodoTxOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testMockStorage := mockStorage.NewMockStorage(ctrl)
	var capturedTodoID int64
	testMockStorage.EXPECT().
		CreateTodoDirectory(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, todoID int64) {
			capturedTodoID = todoID
		}).
		Times(1)

	todoTitle := util.RandomString(10)

	result, err := testStore.CreateTodoTx(context.Background(), CreateTodoTxParams{
		TodoTitle: todoTitle,
		Storage:   testMockStorage,
	})
	require.NoError(t, err)

	// Query the db to find the row
	actualTodo, err := testStore.GetTodo(context.Background(), result.Todo.ID)
	require.NoError(t, err)
	require.Equal(t, result.Todo, actualTodo)

	// Check todoID called in storage
	require.Equal(t, actualTodo.ID, capturedTodoID)
}

func TestCreateTodoTxStorageFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testMockStorage := mockStorage.NewMockStorage(ctrl)
	expectedError := errors.New("storage failure")
	var capturedTodoID int64
	testMockStorage.EXPECT().
		CreateTodoDirectory(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, todoID int64) error {
			capturedTodoID = todoID
			return expectedError
		}).
		Times(1)

	todoTitle := util.RandomString(10)

	result, err := testStore.CreateTodoTx(context.Background(), CreateTodoTxParams{
		TodoTitle: todoTitle,
		Storage:   testMockStorage,
	})
	require.Error(t, err)
	require.EqualError(t, err, expectedError.Error())

	// Query the db to find the row
	actualTodo, err := testStore.GetTodo(context.Background(), result.Todo.ID)
	require.EqualError(t, err, ErrRecordNotFound.Error())
	require.Empty(t, actualTodo)

	// Check todoID called in storage
	require.Equal(t, result.Todo.ID, capturedTodoID)
}
