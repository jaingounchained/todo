package localfilenotification

import (
	notification "github.com/jaingounchained/todo/notification"
)

type LocalFileNotication struct{}

func NewLocalFileNotification() notification.NotificationSender {
	return &LocalFileNotication{}
}

// SendNotification implements notification.NotificationSender.
func (l *LocalFileNotication) SendNotification() error {
	panic("unimplemented")
}
