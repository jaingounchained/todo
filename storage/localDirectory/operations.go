package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func (storage *LocalStorage) CreateTodoDirectory(ctx context.Context, todoID int64) error {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)
	if stat, err := os.Stat(todoDirectory); err != nil || stat.IsDir() {
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
