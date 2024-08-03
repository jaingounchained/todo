package storage

import "context"

type Storage interface {
	CreateTodoDirectory(ctx context.Context, todoID int64) error
	DeleteTodoDirectory(ctx context.Context, todoID int64) error
	SaveFile(ctx context.Context, todoID int64, fileName string, byte []byte) error
	DeleteFile(ctx context.Context, todoID int64, fileName string) error
	GetFileContents(ctx context.Context, todoID int64, fileName string) ([]byte, error)
}
