package applenotification

import (
	notification "github.com/jaingounchained/todo/notification"
)

type AppleAPNSender struct{}

func NewAppleAPNSender() notification.NotificationSender {
	return &AppleAPNSender{}
}

// SendNotification implements notification.NotificationSender.
func (a *AppleAPNSender) SendNotification() error {
	panic("unimplemented")
}
