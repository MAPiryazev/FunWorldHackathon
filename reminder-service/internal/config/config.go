package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// RabbitMQConfig содержит креды RabbitMQ
type RabbitMQConfig struct {
	User     string
	Password string
}

// LoadRabbitMQConfig возвращает конфиг для rabbitmq
func LoadRabbitMQConfig(path string) (*RabbitMQConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("не указан путь до env файла")
	}
	err := godotenv.Load(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки env файла %v", err)
	}

	tempUser := os.Getenv("RABBITMQ_DEFAULT_USER")
	tempPassword := os.Getenv("RABBITMQ_DEFAULT_PASS")

	if tempPassword == "" || tempUser == "" {
		return nil, fmt.Errorf("найдены не все переменные для подключения к rabbit")
	}

	return &RabbitMQConfig{
		User:     tempUser,
		Password: tempPassword,
	}, nil
}
