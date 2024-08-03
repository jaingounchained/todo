package storage

import (
	"errors"
	"os"
)

type LocalStorage struct {
	directoryPath string
}

func New(directoryPath string) (*LocalStorage, error) {
	if stat, err := os.Stat(directoryPath); err != nil || !stat.IsDir() {
		return nil, errors.New("Directory now present")
	}

	return &LocalStorage{
		directoryPath: directoryPath,
	}, nil
}
