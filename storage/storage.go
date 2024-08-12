package storage

import "context"

type FileContents map[string][]byte

type Storage interface {
	CreateTodoDirectory(ctx context.Context, todoID int64) error
	DeleteTodoDirectory(ctx context.Context, todoID int64) error
	SaveFile(ctx context.Context, todoID int64, fileName string, byte []byte) error
	SaveMultipleFilesSafely(ctx context.Context, todoID int64, fileContents FileContents) error
	DeleteFile(ctx context.Context, todoID int64, fileName string) error
	// TODO: Pass a byte array rather than returning it, for better performance
	GetFileContents(ctx context.Context, todoID int64, fileName string) ([]byte, error)
	CloseConnection(ctx context.Context)
}
