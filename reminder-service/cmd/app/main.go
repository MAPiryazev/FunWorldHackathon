package main

import (
	"log"
	"os"

	"reminder-service/internal/config"
	"reminder-service/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Определяем путь к .env файлу
	// В Docker контейнере используем /app/.env, локально - ../../../.env
	envPath := "/app/.env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		envPath = "../../../.env" // для локальной разработки
	}

	// Инициализируем все сервисы
	notificationService, err := server.InitNotificationService(envPath)
	if err != nil {
		log.Fatalf("[main] Не удалось инициализировать сервисы: %v", err)
	}

	// Получаем адрес сервера из переменной окружения или используем дефолтный
	apiCfg, err := config.LoadAPIConfig(envPath)
	if err != nil {
		log.Printf("[main] Предупреждение: не удалось загрузить конфиг API (%v), используем дефолтные значения", err)
		apiCfg = &config.APIConfig{
			Host: "0.0.0.0",
			Port: "8080",
		}
	}

	log.Printf("[main] Запуск reminder-service на %s", ":"+apiCfg.Port)

	// Создаем и запускаем сервер
	srv := server.NewServer(":"+apiCfg.Port, notificationService)
	srv.Start()
}
