package notification

type NotificationSender interface {
	SendNotification() error
}
