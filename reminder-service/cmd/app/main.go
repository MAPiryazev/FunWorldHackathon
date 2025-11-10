package main

import (
	"log"
	"os"
	"path/filepath"

	"reminder-service/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Определяем путь к .env файлу
	// Можно передать через переменную окружения ENV_PATH или использовать дефолтный путь
	envPath := os.Getenv("ENV_PATH")
	if envPath == "" {
		// Пытаемся найти .env в корне проекта (относительно cmd/app/main.go это ../../.env)
		// Или в текущей директории
		possiblePaths := []string{
			"../../.env",
			"../.env",
			".env",
			"./.env",
		}

		for _, path := range possiblePaths {
			if absPath, err := filepath.Abs(path); err == nil {
				if _, err := os.Stat(absPath); err == nil {
					envPath = absPath
					log.Printf("[main] Используется .env файл: %s", envPath)
					break
				}
			}
		}

		if envPath == "" {
			log.Fatal("[main] Не найден .env файл. Установите ENV_PATH или поместите .env в корень проекта")
		}
	}

	// Инициализируем все сервисы
	notificationService, err := server.InitNotificationService(envPath)
	if err != nil {
		log.Fatalf("[main] Не удалось инициализировать сервисы: %v", err)
	}

	// Получаем адрес сервера из переменной окружения или используем дефолтный
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("[main] Запуск reminder-service на %s", addr)

	// Создаем и запускаем сервер
	srv := server.NewServer(addr, notificationService)
	srv.Start()
}
