package db

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	storage "github.com/jaingounchained/todo/storage"
	mockStorage "github.com/jaingounchained/todo/storage/mock"
	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

// TODO: Improve tests by using anonymous struct
func TestUploadAttachmentTxOK(t *testing.T) {
	// Setup
	// Insert a todo in DB
	todo := createRandomTodo(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileContentMap := make(map[string][]byte)
	expectedFileNames := make([]string, 0)
	expectedFileContents := make([][]byte, 0)
	// Insert 3 attachment for the todo
	n := 3
	for i := 0; i < n; i++ {
		fileName, fileContents := util.RandomString(10), []byte(util.RandomString(1))
		fileContentMap[fileName] = fileContents
		expectedFileNames = append(expectedFileNames, fileName)
		expectedFileContents = append(expectedFileContents, fileContents)
	}

	testMockStorage := mockStorage.NewMockStorage(ctrl)
	capturedFileContents := make(map[string][]byte)
	testMockStorage.EXPECT().
		SaveMultipleFilesSafely(gomock.Any(), gomock.Eq(todo.ID), gomock.Any()).
		Do(func(_ context.Context, _ int64, fileContents storage.FileContents) {
			capturedFileContents = fileContents
		}).
		Times(1)

	err := testStore.UploadAttachmentTx(context.Background(), UploadAttachmentTxParams{
		Todo:         todo,
		FileContents: fileContentMap,
		Storage:      testMockStorage,
	})
	// Increment todo filecount in memory
	todo.FileCount += int32(n)
	require.NoError(t, err)

	// Query the db to find the todo
	updatedTodo, err := testStore.GetTodo(context.Background(), todo.ID)
	require.NoError(t, err)
	compareTodos(t, updatedTodo, todo)

	// Query the db to find the attachments, of the todo
	actualAttachments, err := testStore.ListAttachmentOfTodo(context.Background(), todo.ID)
	require.NoError(t, err)
	require.Len(t, actualAttachments, n)
	for _, actualAttachment := range actualAttachments {
		require.Contains(t, expectedFileNames, actualAttachment.OriginalFilename)
	}

	// Check the contents of mock storage call
	require.Len(t, capturedFileContents, n)
	for _, contents := range capturedFileContents {
		require.Contains(t, expectedFileContents, contents)
	}
}

func TestUploadAttachmentTxStorageFailure(t *testing.T) {
	// Setup
	// Insert a todo in DB
	todo := createRandomTodo(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileContentMap := make(map[string][]byte)
	expectedFileNames := make([]string, 0)
	expectedFileContents := make([][]byte, 0)
	// Insert 3 attachment for the todo
	n := 3
	for i := 0; i < n; i++ {
		fileName, fileContents := util.RandomString(10), []byte(util.RandomString(1))
		fileContentMap[fileName] = fileContents
		expectedFileNames = append(expectedFileNames, fileName)
		expectedFileContents = append(expectedFileContents, fileContents)
	}

	testMockStorage := mockStorage.NewMockStorage(ctrl)
	storageError := errors.New("storage failure")
	capturedFileContents := make(map[string][]byte)
	testMockStorage.EXPECT().
		SaveMultipleFilesSafely(gomock.Any(), gomock.Eq(todo.ID), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, fileContents storage.FileContents) error {
			capturedFileContents = fileContents
			return storageError
		}).
		Times(1)

	err := testStore.UploadAttachmentTx(context.Background(), UploadAttachmentTxParams{
		Todo:         todo,
		FileContents: fileContentMap,
		Storage:      testMockStorage,
	})
	require.Error(t, err)
	require.EqualError(t, err, storageError.Error())

	// Query the db to find the todo
	updatedTodo, err := testStore.GetTodo(context.Background(), todo.ID)
	require.NoError(t, err)
	compareTodos(t, updatedTodo, todo)

	// Query the db to find the attachments, of the todo
	actualAttachments, err := testStore.ListAttachmentOfTodo(context.Background(), todo.ID)
	require.Len(t, actualAttachments, 0)

	// Check the contents of mock storage call
	require.Len(t, capturedFileContents, n)
	for _, contents := range capturedFileContents {
		require.Contains(t, expectedFileContents, contents)
	}
}
