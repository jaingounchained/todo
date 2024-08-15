package storage

import (
	"errors"

	"github.com/jaingounchained/todo/util"
)

type LocalStorage struct {
	directoryPath string
}

func New(directoryPath string) (*LocalStorage, error) {
	if !util.DirExists(directoryPath) {
		return nil, errors.New("Directory doesn't exist")
	}

	return &LocalStorage{
		directoryPath: directoryPath,
	}, nil
}
