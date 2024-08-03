package storage

import "context"

type FileContents map[string][]byte

type Storage interface {
	CreateTodoDirectory(ctx context.Context, todoID int64) error
	DeleteTodoDirectory(ctx context.Context, todoID int64) error
	SaveFile(ctx context.Context, todoID int64, fileName string, byte []byte) error
	SaveMultipleFilesSafely(ctx context.Context, todoID int64, fileContents FileContents) error
	DeleteFile(ctx context.Context, todoID int64, fileName string) error
	GetFileContents(ctx context.Context, todoID int64, fileName string) ([]byte, error)
}
