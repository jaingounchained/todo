package db

import (
	"context"
	"testing"
	"time"

	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

func createRandomAttachmentForTodo(t *testing.T, todo Todo) Attachment {
	arg := CreateAttachmentParams{
		TodoID:           todo.ID,
		OriginalFilename: util.RandomString(10),
		StorageFilename:  util.RandomString(10),
	}

	attachment, err := testStore.CreateAttachment(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, attachment)

	require.Equal(t, arg.TodoID, attachment.TodoID)
	require.Equal(t, arg.OriginalFilename, attachment.OriginalFilename)
	require.Equal(t, arg.StorageFilename, attachment.StorageFilename)

	require.NotZero(t, attachment.ID)
	require.NotZero(t, attachment.CreatedAt)

	return attachment
}

func compareAttachment(t *testing.T, attachment1, attachment2 Attachment) {
	require.Equal(t, attachment1.ID, attachment2.ID)
	require.Equal(t, attachment1.TodoID, attachment2.TodoID)
	require.Equal(t, attachment1.OriginalFilename, attachment2.OriginalFilename)
	require.Equal(t, attachment1.StorageFilename, attachment2.StorageFilename)
	require.WithinDuration(t, attachment1.CreatedAt, attachment2.CreatedAt, time.Second)
}

func TestCreateAttachment(t *testing.T) {
	todo := createRandomTodo(t)
	createRandomAttachmentForTodo(t, todo)
}

func TestGetAttachment(t *testing.T) {
	todo := createRandomTodo(t)
	attachment1 := createRandomAttachmentForTodo(t, todo)

	attachment2, err := testStore.GetAttachment(context.Background(), attachment1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, attachment2)

	compareAttachment(t, attachment1, attachment2)
}

func TestListAttachmentOfTodo(t *testing.T) {
	todo := createRandomTodo(t)
	attachment1 := createRandomAttachmentForTodo(t, todo)
	attachment2 := createRandomAttachmentForTodo(t, todo)
	attachment3 := createRandomAttachmentForTodo(t, todo)

	attachments, err := testStore.ListAttachmentOfTodo(context.Background(), todo.ID)
	require.NoError(t, err)
	require.NotEmpty(t, attachment2)

	compareAttachment(t, attachment1, attachments[0])
	compareAttachment(t, attachment2, attachments[1])
	compareAttachment(t, attachment3, attachments[2])
}

func TestDeleteAttachment(t *testing.T) {
	todo := createRandomTodo(t)
	attachment1 := createRandomAttachmentForTodo(t, todo)

	err := testStore.DeleteAttachment(context.Background(), attachment1.ID)
	require.NoError(t, err)

	attachment2, err := testStore.GetAttachment(context.Background(), attachment1.ID)
	require.Error(t, err)
	require.EqualError(t, err, ErrRecordNotFound.Error())
	require.Empty(t, attachment2)
}

func TestDeleteAttachmentsOfTodo(t *testing.T) {
	todo := createRandomTodo(t)
	attachments := make([]Attachment, 0)
	for i := 0; i < 3; i++ {
		attachments = append(attachments, createRandomAttachmentForTodo(t, todo))
	}

	err := testStore.DeleteAttachmentsOfTodo(context.Background(), todo.ID)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		attachment, err := testStore.GetAttachment(context.Background(), attachments[i].ID)
		require.Error(t, err)
		require.EqualError(t, err, ErrRecordNotFound.Error())
		require.Empty(t, attachment)
	}
}
