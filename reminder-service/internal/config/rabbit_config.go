package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// RabbitMQConfig содержит креды и адрес RabbitMQ
type RabbitMQConfig struct {
	User     string
	Password string
	Host     string
	Port     string
}

// LoadRabbitMQConfig загружает конфигурацию RabbitMQ из ENV
func LoadRabbitMQConfig(path string) (*RabbitMQConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("не указан путь до env файла")
	}

	// загружаем env
	if err := godotenv.Load(path); err != nil {
		return nil, fmt.Errorf("ошибка загрузки env файла: %v", err)
	}

	user := os.Getenv("RABBITMQ_DEFAULT_USER")
	password := os.Getenv("RABBITMQ_DEFAULT_PASS")
	host := os.Getenv("RABBITMQ_HOST")
	port := os.Getenv("RABBITMQ_PORT")

	if user == "" || password == "" || host == "" || port == "" {
		return nil, fmt.Errorf("не найдены все переменные для подключения к RabbitMQ")
	}

	return &RabbitMQConfig{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
	}, nil
}
