package notifier

import (
	"context"
	"fmt"
	"net/smtp"

	"L3.1/internal/models"
)

// GmailNotifier хранит данные пользователя для уведомлений через Gmail
type GmailNotifier struct {
	from     string
	password string
}

// NewGmailNotifier создает  GmailNotifier
func NewGmailNotifier(cfgFrom, cfgPassword string) (*GmailNotifier, error) {
	if cfgFrom == "" || cfgPassword == "" {
		return nil, fmt.Errorf("поля from и password не должны быть nil")
	}
	return &GmailNotifier{
		from:     cfgFrom,
		password: cfgPassword,
	}, nil
}

// Send отправляет уведомления через Gmail SMTP
func (gn *GmailNotifier) Send(ctx context.Context, msg *models.RabbitMQMessage) error {
	to := []string{msg.To}
	body := fmt.Sprintf("Subject: %s\r\n\r\n%s", msg.Subject, msg.Body)

	auth := smtp.PlainAuth("", gn.from, gn.password, "smtp.gmail.com")

	err := smtp.SendMail("smtp.gmail.com:587", auth, gn.from, to, []byte(body))
	if err != nil {
		return fmt.Errorf("ошибка при отправке письма: %w", err)
	}

	return nil
}
