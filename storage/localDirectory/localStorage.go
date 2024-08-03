package storage

import (
	"errors"
	"os"
)

type LocalStorage struct {
	directoryPath string
}

func New(directoryPath string) (*LocalStorage, error) {
	if _, err := os.Stat(directoryPath); os.IsNotExist(err) {
		return nil, errors.New("Directory now present")
	}

	return &LocalStorage{
		directoryPath: directoryPath,
	}, nil
}
