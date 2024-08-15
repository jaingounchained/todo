package db

import (
	"context"
	"testing"
	"time"

	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

func createRandomTodo(t *testing.T) Todo {
	user := createRandomUser(t)

	arg := CreateTodoParams{
		Owner: user.Username,
		Title: util.RandomString(10),
	}

	todo, err := testStore.CreateTodo(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, todo)

	require.Equal(t, arg.Owner, todo.Owner)
	require.Equal(t, arg.Title, todo.Title)

	require.NotZero(t, todo.ID)
	require.Equal(t, todo.FileCount, int32(0))
	require.Equal(t, todo.Status, "incomplete")
	require.NotZero(t, todo.CreatedAt)

	return todo
}

func compareTodos(t *testing.T, todo1, todo2 Todo) {
	require.Equal(t, todo1.ID, todo2.ID)
	require.Equal(t, todo1.Owner, todo2.Owner)
	require.Equal(t, todo1.Title, todo2.Title)
	require.Equal(t, todo1.Status, todo2.Status)
	require.Equal(t, todo1.FileCount, todo2.FileCount)
	require.WithinDuration(t, todo1.CreatedAt, todo2.CreatedAt, time.Second)
}

func TestCreateTodo(t *testing.T) {
	createRandomTodo(t)
}

func TestGetTodo(t *testing.T) {
	todo1 := createRandomTodo(t)
	todo2, err := testStore.GetTodo(context.Background(), todo1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, todo2)

	compareTodos(t, todo1, todo2)
}

func TestUpdateTodoTitleStatus(t *testing.T) {
	tcs := []struct {
		name          string
		todo          Todo
		updatedTitle  *string
		updatedStatus *string
	}{
		{
			name:          "UpdateTitleAndStatus",
			todo:          createRandomTodo(t),
			updatedTitle:  util.RandomStringPointer(10),
			updatedStatus: util.RandomStatusPointer(),
		},
		{
			name:          "UpdateTitleOnly",
			todo:          createRandomTodo(t),
			updatedTitle:  util.RandomStringPointer(10),
			updatedStatus: nil,
		},
		{
			name:          "UpdateStatusOnly",
			todo:          createRandomTodo(t),
			updatedTitle:  nil,
			updatedStatus: util.RandomStatusPointer(),
		},
		{
			name:          "NoUpdate",
			todo:          createRandomTodo(t),
			updatedTitle:  nil,
			updatedStatus: nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			arg := UpdateTodoTitleStatusParams{
				ID:     tc.todo.ID,
				Title:  tc.updatedTitle,
				Status: tc.updatedStatus,
			}

			todo2, err := testStore.UpdateTodoTitleStatus(context.Background(), arg)

			require.NoError(t, err)
			require.NotEmpty(t, todo2)

			if tc.updatedTitle != nil {
				tc.todo.Title = *tc.updatedTitle
			}

			if tc.updatedStatus != nil {
				tc.todo.Status = *tc.updatedStatus
			}

			compareTodos(t, todo2, tc.todo)
		})
	}
}

func TestUpdateTodoFileCount(t *testing.T) {
	todo1 := createRandomTodo(t)

	updatedFileCount := int32(util.RandomInt(-10, 10))
	arg := UpdateTodoFileCountParams{
		ID:        todo1.ID,
		FileCount: updatedFileCount,
	}

	todo2, err := testStore.UpdateTodoFileCount(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, todo2)

	todo1.FileCount = todo1.FileCount + int32(updatedFileCount)
	compareTodos(t, todo1, todo2)
}

func TestDeleteTodo(t *testing.T) {
	todo1 := createRandomTodo(t)
	err := testStore.DeleteTodo(context.Background(), todo1.ID)
	require.NoError(t, err)

	todo2, err := testStore.GetTodo(context.Background(), todo1.ID)
	require.Error(t, err)
	require.EqualError(t, err, ErrRecordNotFound.Error())
	require.Empty(t, todo2)
}

func TestListTodos(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomTodo(t)
	}

	arg := ListTodosParams{
		Limit:  5,
		Offset: 5,
	}

	todos, err := testStore.ListTodos(context.Background(), arg)
	require.NoError(t, err)
	require.Len(t, todos, 5)

	for _, todo := range todos {
		require.NotEmpty(t, todo)
	}
}
