package notification

type NotificationService interface {
	SendNotification(message string) error
}

type NullNotificationService struct {
}

func NewNullNotificationService() NotificationService {
	return &NullNotificationService{}
}

func (s *NullNotificationService) SendNotification(message string) error {
	return nil
}
