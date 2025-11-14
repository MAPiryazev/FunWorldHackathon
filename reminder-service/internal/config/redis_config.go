package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// RedisConfig содержит креды для Redis
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// LoadRedisConfig загружает конфигурацию Redis из ENV
func LoadRedisConfig(path string) (*RedisConfig, error) {
	if path == "" {
		return nil, fmt.Errorf("не указан путь до env файла")
	}

	// загружаем env
	if err := godotenv.Load(path); err != nil {
		return nil, fmt.Errorf("ошибка загрузки env файла: %v", err)
	}

	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	password := os.Getenv("REDIS_PASSWORD")

	db := 0
	if dbStr := os.Getenv("REDIS_DB"); dbStr != "" {
		if v, err := strconv.Atoi(dbStr); err == nil {
			db = v
		}
	}

	return &RedisConfig{
		Host:     host,
		Port:     port,
		Password: password,
		DB:       db,
	}, nil
}
