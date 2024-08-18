package localstorage

import (
	"fmt"
)

type LocalDirectoryForTodoAlreadyExistError error

func newLocalDirectoryForTodoAlreadyExistError(todo int64) LocalDirectoryForTodoAlreadyExistError {
	return fmt.Errorf("Local directory already exist for the todo: %d", todo)
}

type LocalDirectoryForTodoDoesNotExistError error

func newLocalDirectoryForTodoDoesNotExistError(todo int64) LocalDirectoryForTodoDoesNotExistError {
	return fmt.Errorf("Local directory does not exist for the todo: %d", todo)
}

type FileAlreadyExistForTheTodoError error

func newFileAlreadyExistForTheTodoError(todo int64, filename string) FileAlreadyExistForTheTodoError {
	return fmt.Errorf("Filename: %s already exist for the todo: %d", filename, todo)
}

type FileDoesNotExistForTheTodoError error

func newFileDoesNotExistForTheTodoError(todo int64, filename string) FileDoesNotExistForTheTodoError {
	return fmt.Errorf("Filename: %s does not exist for the todo: %d", filename, todo)
}
