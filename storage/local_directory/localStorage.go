package localstorage

import (
	"errors"

	storage "github.com/jaingounchained/todo/storage"
	"github.com/jaingounchained/todo/util"
)

type LocalStorage struct {
	directoryPath string
}

func NewLocalStorage(directoryPath string) (storage.Storage, error) {
	if !util.DirExists(directoryPath) {
		return nil, errors.New("Directory doesn't exist")
	}

	return &LocalStorage{
		directoryPath: directoryPath,
	}, nil
}
