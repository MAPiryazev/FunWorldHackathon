package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"L3.1/internal/config"
	"L3.1/internal/handlers"
	"L3.1/internal/infrastructure"
	"L3.1/internal/notifier"
	"L3.1/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server представляет HTTP сервер
type Server struct {
	httpServer      *http.Server
	notificationSvc *service.NotificationService
}

// InitNotificationService инициализация
func InitNotificationService() (*service.NotificationService, error) {
	rabbitCfg, err := config.LoadRabbitMQConfig("../../environment/.env")
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфига RabbitMQ: %w", err)
	}

	redisClient := infrastructure.NewRedisClient("localhost:6379", "", 0)

	rabbitMQClient, err := infrastructure.NewRabbitMQClient(*rabbitCfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к RabbitMQ: %w", err)
	}

	gmailConfig, err := config.LoadGmaiLConfig("../../environment/.env")
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфига Gmail: %w", err)
	}

	gmailNotifier, err := notifier.NewGmailNotifier(gmailConfig.From, gmailConfig.Password)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания Gmail нотификатора: %w", err)
	}

	notificationService, err := service.NewNotificationService(
		redisClient,
		rabbitMQClient,
		gmailNotifier,
		"delayed_queue",
		"ready_queue",
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервиса: %w", err)
	}

	return notificationService, nil
}

// NewServer создает новый сервер
func NewServer(addr string, notificationSvc *service.NotificationService) *Server {
	handler := handlers.NewDefaultHandler(notificationSvc)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Route("/notify", func(r chi.Router) {
		r.Post("/", handler.CreateNotificationHandler)
		r.Get("/status", handler.GetStatusHandler)
		r.Delete("/cancel", handler.CancelNotificationHandler)
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	return &Server{
		httpServer:      srv,
		notificationSvc: notificationSvc,
	}
}

// Start запускает сервер и фоновые воркеры
func (s *Server) Start() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		log.Println("Запуск Delayed Worker...")
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Delayed Worker упал: %v", r)
			}
		}()
		if err := s.notificationSvc.StartDelayedWorker(ctx); err != nil && err != context.Canceled {
			log.Printf("Ошибка delayed воркера: %v", err)
		}
	}()

	go func() {
		log.Println("Запуск Ready Worker...")
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Ready Worker упал: %v", r)
			}
		}()
		if err := s.notificationSvc.StartWorker(ctx); err != nil && err != context.Canceled {
			log.Printf("Ошибка ready воркера: %v", err)
		}
	}()

	// Запускаем HTTP сервер
	go func() {
		log.Printf("Сервер запущен на порту %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ошибка запуска сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Завершение работы сервера")
	cancel() // отменяем контекст для воркеров
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("ошибка при остановке сервера: %v", err)
	}

	log.Println("Сервер остановлен корректно")
}
