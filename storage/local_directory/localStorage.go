package storage

import (
	"errors"

	"github.com/jaingounchained/todo/util"
	"go.uber.org/zap"
)

type LocalStorage struct {
	directoryPath string
	logger        *zap.Logger
}

func New(logger *zap.Logger, directoryPath string) (*LocalStorage, error) {
	if !util.DirExists(directoryPath) {
		return nil, errors.New("Directory doesn't exist")
	}

	return &LocalStorage{
		directoryPath: directoryPath,
		logger:        logger,
	}, nil
}
