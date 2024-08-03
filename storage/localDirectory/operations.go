package storage

import (
	"context"
	"errors"
	"fmt"
	storage "github.com/jaingounchained/todo/storage"
	"os"
	"path/filepath"
)

func (storage *LocalStorage) CreateTodoDirectory(ctx context.Context, todoID int64) error {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)
	fmt.Println(todoDirectory)
	if _, err := os.Stat(todoDirectory); os.IsExist(err) {
		return errors.New("Directory already present")
	}

	return os.Mkdir(todoDirectory, 0755)
}

func (storage *LocalStorage) DeleteTodoDirectory(ctx context.Context, todoID int64) error {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)

	return os.RemoveAll(todoDirectory)
}

func (storage *LocalStorage) SaveFile(ctx context.Context, todoID int64, fileName string, byte []byte) error {
	todoFilePath := storage.todoFilePath(todoID, fileName)

	return os.WriteFile(todoFilePath, byte, 0644)
}

func (storage *LocalStorage) DeleteFile(ctx context.Context, todoID int64, fileName string) error {
	todoFilePath := storage.todoFilePath(todoID, fileName)

	return os.Remove(todoFilePath)
}

func (storage *LocalStorage) GetFileContents(ctx context.Context, todoID int64, fileName string) ([]byte, error) {
	todoFilePath := filepath.Join(storage.directoryPath, todoDirectoryName(todoID), fileName)

	return os.ReadFile(todoFilePath)
}

func (storage *LocalStorage) todoAbsoluteDirectory(todoID int64) string {
	return filepath.Join(storage.directoryPath, todoDirectoryName(todoID))
}

func (storage *LocalStorage) todoFilePath(todoID int64, fileName string) string {
	return filepath.Join(storage.todoAbsoluteDirectory(todoID), fileName)
}

func todoDirectoryName(todoID int64) string {
	return fmt.Sprintf("%08d", todoID)
}

// TODO: optimize this function
func (storage *LocalStorage) SaveMultipleFilesSafely(ctx context.Context, todoID int64, fileContents storage.FileContents) error {
	// Temporary file storage
	tempFiles := make([]*os.File, len(fileContents))
	fileNames := make([]string, 0, len(fileContents))

	// Create and write to temporary files
	i := 0
	for name, data := range fileContents {
		tempFile, err := os.CreateTemp("temp", "example")
		if err != nil {
			cleanup(tempFiles)
			fmt.Printf("Failed to create temp file for %s: %v\n", name, err)
			return err
		}
		defer tempFile.Close()

		if _, err := tempFile.Write(data); err != nil {
			cleanup(tempFiles)
			fmt.Printf("Failed to write to temp file for %s: %v\n", name, err)
		}

		tempFiles[i] = tempFile
		fileNames = append(fileNames, storage.todoFilePath(todoID, name))
		i++
	}

	// Rename all temporary files to final names
	for i, tempFile := range tempFiles {
		finalName := fileNames[i]
		if err := os.Rename(tempFile.Name(), finalName); err != nil {
			fmt.Printf("Failed to rename temp file to %s: %v\n", finalName, err)
			revertRenames(tempFiles, fileNames)
			cleanup(tempFiles)
			fmt.Printf("Failed to complete all renames; changes reverted.\n")
			return err
		}
	}

	return nil
}

func cleanup(files []*os.File) {
	for _, file := range files {
		if file != nil {
			os.Remove(file.Name())
		}
	}
}

func revertRenames(tempFiles []*os.File, fileNames []string) {
	for i, file := range tempFiles {
		finalName := fileNames[i]
		if _, err := os.Stat(finalName); err == nil {
			if err := os.Rename(finalName, file.Name()); err != nil {
				fmt.Printf("Failed to revert rename for %s: %v\n", finalName, err)
			}
		}
	}
}
