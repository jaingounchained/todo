package util

import (
	"fmt"
	"time"
)

func ConvertDurationToCron(duration time.Duration) (string, error) {
	seconds := int(duration.Seconds())

	switch {
	case seconds < 60:
		return fmt.Sprintf("*/%d * * * * *", seconds), nil // Every N seconds
	case seconds < 3600:
		minutes := seconds / 60
		return fmt.Sprintf("*/%d * * * *", minutes), nil // Every N minutes
	case seconds < 86400:
		hours := seconds / 3600
		return fmt.Sprintf("0 */%d * * *", hours), nil // Every N hours
	case seconds < 604800:
		days := seconds / 86400
		return fmt.Sprintf("0 0 */%d * *", days), nil // Every N days
	default:
		return "", fmt.Errorf("duration too large or unsupported for conversion to cron expression")
	}
}
