package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// APIConfig содержит креды для API
type APIConfig struct {
	Host string
	Port string
}

// LoadAPIConfig загружает конфигурацию API из ENV
func LoadAPIConfig(path string) (*APIConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("не указан путь до env файла")
	}

	if err := godotenv.Load(path); err != nil {
		return nil, fmt.Errorf("ошибка загрузки env файла: %v", err)
	}

	host := os.Getenv("API_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	return &APIConfig{
		Host: host,
		Port: port,
	}, nil
}
