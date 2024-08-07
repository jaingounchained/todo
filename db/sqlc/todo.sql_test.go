package db

import (
	"context"
	"testing"
	"time"

	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

func createRandomTodo(t *testing.T) Todo {
	title := util.RandomString(10)

	todo, err := testStore.CreateTodo(context.Background(), title)
	require.NoError(t, err)
	require.NotEmpty(t, todo)

	require.Equal(t, todo.Title, title)

	require.NotZero(t, todo.ID)
	require.Equal(t, todo.FileCount, int32(0))
	require.Equal(t, todo.Status, "incomplete")
	require.NotZero(t, todo.CreatedAt)

	return todo
}

func compareTodos(t *testing.T, todo1, todo2 Todo) {
	require.Equal(t, todo1.ID, todo2.ID)
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

func TestUpdateTodoTitle(t *testing.T) {
	todo1 := createRandomTodo(t)

	updatedTitle := util.RandomString(10)
	arg := UpdateTodoTitleParams{
		ID:    todo1.ID,
		Title: updatedTitle,
	}

	todo2, err := testStore.UpdateTodoTitle(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, todo2)

	todo1.Title = updatedTitle
	compareTodos(t, todo1, todo2)
}

func TestUpdateTodoStatus(t *testing.T) {
	todo1 := createRandomTodo(t)

	updatedStatus := "complete"
	arg := UpdateTodoStatusParams{
		ID:     todo1.ID,
		Status: updatedStatus,
	}

	todo2, err := testStore.UpdateTodoStatus(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, todo2)

	todo1.Status = updatedStatus
	compareTodos(t, todo1, todo2)
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
