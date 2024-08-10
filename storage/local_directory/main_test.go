package storage

import (
	"os"
	"path/filepath"
	"testing"
)

var localStorageTest *LocalStorage

func TestMain(m *testing.M) {
	// Setup
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	testLocalDirectoryPath := filepath.Join(cwd, "test-local-storage")

	err = os.MkdirAll(testLocalDirectoryPath, 0755)
	if err != nil {
		os.Exit(1)
	}

	localStorageTest, err = New(nil, testLocalDirectoryPath)
	if err != nil {
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Teardown
	err = os.RemoveAll(testLocalDirectoryPath)
	if err != nil {
		os.Exit(1)
	}

	localStorageTest = nil
	os.Exit(code)
}
