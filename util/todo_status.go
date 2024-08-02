package util

const (
	complete   = "complete"
	incomplete = "incomplete"
)

func IsSupportedTodoStatus(todoStatus string) bool {
	switch todoStatus {
	case complete, incomplete:
		return true
	}

	return false
}
