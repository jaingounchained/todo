package localstorage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

func createTodoDirectory(t *testing.T) (int64, string) {
	todoID := util.RandomInt(1, 100000)
	expectedTodoDir := filepath.Join(localStorageTest.directoryPath, fmt.Sprintf("%08d", todoID))

	// Attempt to make a directory
	err := localStorageTest.CreateTodoDirectory(context.Background(), todoID)
	require.NoError(t, err)
	require.DirExists(t, expectedTodoDir)

	return todoID, expectedTodoDir
}

func TestCreateTodoDirectory(t *testing.T) {
	todoID, _ := createTodoDirectory(t)

	// Attempt to make the same directory again
	err := localStorageTest.CreateTodoDirectory(context.Background(), todoID)
	require.Error(t, err)
}

func TestDeleteTodoDirectory(t *testing.T) {
	todoID, expectedTodoDir := createTodoDirectory(t)

	// Attempt to delete the directory
	err := localStorageTest.DeleteTodoDirectory(context.Background(), todoID)
	require.NoError(t, err)
	require.NoDirExists(t, expectedTodoDir)

	// Attempt to delete the same directory again
	err = localStorageTest.DeleteTodoDirectory(context.Background(), todoID)
	require.Error(t, err)
}

func TestSaveFile(t *testing.T) {
	todoID, expectedTodoDir := createTodoDirectory(t)
	fileName, fileContents := util.RandomString(10), []byte(util.RandomString(100))
	expectedFilePath := filepath.Join(expectedTodoDir, fileName)

	// Attempt to create a file
	err := localStorageTest.SaveFile(context.Background(), todoID, fileName, fileContents)
	require.NoError(t, err)
	require.FileExists(t, expectedFilePath)

	// Attempt to save the same file again
	err = localStorageTest.SaveFile(context.Background(), todoID, fileName, fileContents)
	require.Error(t, err)
}

func TestDeleteFile(t *testing.T) {
	todoID, expectedTodoDir := createTodoDirectory(t)

	fileName, fileContents := util.RandomString(10), []byte(util.RandomString(100))
	expectedFilePath := filepath.Join(expectedTodoDir, fileName)

	// Attempt to create the file
	err := localStorageTest.SaveFile(context.Background(), todoID, fileName, fileContents)
	require.NoError(t, err)
	require.FileExists(t, expectedFilePath)

	// Attempt to delete the file
	err = localStorageTest.DeleteFile(context.Background(), todoID, fileName)
	require.NoError(t, err)

	// Attempt to delete the file again
	err = localStorageTest.DeleteFile(context.Background(), todoID, fileName)
	require.Error(t, err)
}

func TestGetFileContents(t *testing.T) {
	todoID, expectedTodoDir := createTodoDirectory(t)
	fileName, fileContents := util.RandomString(10), []byte(util.RandomString(100))
	expectedFilePath := filepath.Join(expectedTodoDir, fileName)

	// Attempt to read the file before it is created
	_, err := localStorageTest.GetFileContents(context.Background(), todoID, fileName)
	require.Error(t, err)

	// Attempt to create the file
	err = localStorageTest.SaveFile(context.Background(), todoID, fileName, fileContents)
	require.NoError(t, err)
	require.FileExists(t, expectedFilePath)

	// Attempt to read the file
	bytes, err := localStorageTest.GetFileContents(context.Background(), todoID, fileName)
	require.NoError(t, err)
	require.Equal(t, bytes, fileContents)
}

func TestSaveMultipleFilesSafely(t *testing.T) {
	todoID, expectedTodoDir := createTodoDirectory(t)
	inputFileContents := make(map[string][]byte)
	expectedFilePaths := make(map[string]string)
	for i := 0; i < 5; i++ {
		fileName, bytes := util.RandomString(10), []byte(util.RandomString(100))

		expectedFilePath := filepath.Join(expectedTodoDir, fileName)
		expectedFilePaths[fileName] = expectedFilePath

		inputFileContents[fileName] = bytes
	}

	// Attempt to create multiple files
	err := localStorageTest.SaveMultipleFilesSafely(context.Background(), todoID, inputFileContents)
	require.NoError(t, err)
	for filePath, contents := range inputFileContents {
		require.FileExists(t, expectedFilePaths[filePath])
		bytes, err := localStorageTest.GetFileContents(context.Background(), todoID, filePath)
		require.NoError(t, err)
		require.Equal(t, bytes, contents)
	}
}
