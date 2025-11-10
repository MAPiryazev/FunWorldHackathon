package main

import (
	"log"

	"reminder-service/internal/config"
	"reminder-service/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Инициализируем все сервисы
	notificationService, err := server.InitNotificationService("../../../.env")
	if err != nil {
		log.Fatalf("[main] Не удалось инициализировать сервисы: %v", err)
	}

	// Получаем адрес сервера из переменной окружения или используем дефолтный
	apiCfg, err := config.LoadAPIConfig("../../../.env")

	log.Printf("[main] Запуск reminder-service на %s", ":"+apiCfg.Port)

	// Создаем и запускаем сервер
	srv := server.NewServer(":"+apiCfg.Port, notificationService)
	srv.Start()
}
