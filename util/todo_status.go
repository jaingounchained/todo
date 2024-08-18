package util

const (
	TodoStatusComplete   = "complete"
	TodoStatusIncomplete = "incomplete"
)

func IsSupportedTodoStatus(todoStatus string) bool {
	switch todoStatus {
	case TodoStatusComplete, TodoStatusIncomplete:
		return true
	}

	return false
}
