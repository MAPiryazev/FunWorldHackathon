package main

import (
	"log"

	"L3.1/internal/server"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Инициализируем все сервисы
	notificationService, err := server.InitNotificationService()
	if err != nil {
		log.Fatalf("Не удалось инициализировать сервисы: %v", err)
	}

	// Создаем и запускаем сервер
	srv := server.NewServer(":8080", notificationService)
	srv.Start()
}
