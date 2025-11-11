package config

import (
	"os"

	"github.com/joho/godotenv"
)

// NotificationConfig содержит конфигурацию для отправки уведомлений
type NotificationConfig struct {
	Endpoint string
}

// LoadNotificationConfig загружает конфигурацию notification sender из ENV
// Ожидаем переменную окружения: API_SENDER_ENDPOINT
func LoadNotificationConfig(path string) (*NotificationConfig, error) {
	// Пытаемся загрузить .env по указанному пути (если файл отсутствует — не критично)
	_ = godotenv.Load(path)

	endpoint := os.Getenv("API_SENDER_ENDPOINT")

	return &NotificationConfig{
		Endpoint: endpoint,
	}, nil
}
