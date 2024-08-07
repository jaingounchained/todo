package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	storage "github.com/jaingounchained/todo/storage"
	"github.com/jaingounchained/todo/util"
)

func (storage *LocalStorage) CreateTodoDirectory(ctx context.Context, todoID int64) error {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)
	if util.DirExists(todoDirectory) {
		return errors.New("Directory already exist")
	}

	return os.Mkdir(todoDirectory, 0755)
}

func (storage *LocalStorage) DeleteTodoDirectory(ctx context.Context, todoID int64) error {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)
	if !util.DirExists(todoDirectory) {
		return errors.New("Directory doesn't exist")
	}

	return os.RemoveAll(todoDirectory)
}

func (storage *LocalStorage) SaveFile(ctx context.Context, todoID int64, fileName string, byte []byte) error {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)
	if !util.DirExists(todoDirectory) {
		return errors.New("Directory doesn't exist")
	}

	todoFilePath := filepath.Join(todoDirectory, fileName)
	if util.FileExists(todoFilePath) {
		return errors.New("File already present")
	}

	return os.WriteFile(todoFilePath, byte, 0644)
}

func (storage *LocalStorage) DeleteFile(ctx context.Context, todoID int64, fileName string) error {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)
	if !util.DirExists(todoDirectory) {
		return errors.New("Directory doesn't exist")
	}

	todoFilePath := filepath.Join(todoDirectory, fileName)
	if !util.FileExists(todoFilePath) {
		return errors.New("File not present")
	}

	return os.Remove(todoFilePath)
}

// TODO: Optimize this function, pass in a byte array instead of returning
func (storage *LocalStorage) GetFileContents(ctx context.Context, todoID int64, fileName string) ([]byte, error) {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)
	if !util.DirExists(todoDirectory) {
		return nil, errors.New("Directory doesn't exist")
	}

	todoFilePath := filepath.Join(todoDirectory, fileName)
	if !util.FileExists(todoFilePath) {
		return nil, errors.New("File not present")
	}

	return os.ReadFile(todoFilePath)
}

// TODO: optimize this function
func (storage *LocalStorage) SaveMultipleFilesSafely(ctx context.Context, todoID int64, fileContents storage.FileContents) error {
	todoDirectory := storage.todoAbsoluteDirectory(todoID)
	if !util.DirExists(todoDirectory) {
		return errors.New("Directory doesn't exist")
	}

	// Temporary file storage
	tempFiles := make([]*os.File, len(fileContents))
	fileNames := make([]string, 0, len(fileContents))

	// Create and write to temporary files
	i := 0
	for name, data := range fileContents {
		tempFile, err := os.CreateTemp("", "example")
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
		fileNames = append(fileNames, filepath.Join(todoDirectory, name))
		i++
	}

	// Rename all temporary files to final names
	for i, tempFile := range tempFiles {
		finalName := fileNames[i]
		if err := moveFile(tempFile.Name(), finalName); err != nil {
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

func moveFile(src, dst string) error {
	inputFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	if _, err := io.Copy(outputFile, inputFile); err != nil {
		return err
	}

	return os.Remove(src)
}

func (storage *LocalStorage) todoAbsoluteDirectory(todoID int64) string {
	return filepath.Join(storage.directoryPath, todoDirectoryName(todoID))
}

func todoDirectoryName(todoID int64) string {
	return fmt.Sprintf("%08d", todoID)
}

func (storage *LocalStorage) CloseConnection(ctx context.Context) {}
