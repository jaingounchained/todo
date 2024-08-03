package util

import (
	"github.com/google/uuid"
)

func GenerateUUID() (string, error) {
	v4UUID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return v4UUID.String(), nil
}
