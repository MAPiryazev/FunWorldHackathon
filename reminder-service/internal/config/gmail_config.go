package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// GmailConfig конфиг для gmail_notifier.go
type GmailConfig struct {
	From     string
	Password string
}

// LoadGmaiLConfig функция которая грузит конфиг из .env файла
func LoadGmaiLConfig(path string) (*GmailConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("укажите путь до env файла")
	}
	err := godotenv.Load(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка при загрузке env файла")
	}
	tempFrom := os.Getenv("GMAIL_SERVICE_FROM")
	tempPassword := os.Getenv("GMAIL_SERVICE_PASSWORD")

	if tempFrom == "" || tempPassword == "" {
		return nil, fmt.Errorf("параметры для Gmail сервиса не найдены")
	}

	return &GmailConfig{
		From:     tempFrom,
		Password: tempPassword,
	}, nil
}
