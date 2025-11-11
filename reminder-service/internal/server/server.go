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

	"reminder-service/internal/config"
	"reminder-service/internal/handlers"
	"reminder-service/internal/infrastructure"
	"reminder-service/internal/sender"
	"reminder-service/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	httpServer      *http.Server
	notificationSvc *service.NotificationService
}

func InitNotificationService(envPath string) (*service.NotificationService, error) {
	rabbitCfg, err := config.LoadRabbitMQConfig(envPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфига RabbitMQ: %w", err)
	}

	redisCfg, err := config.LoadRedisConfig(envPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки конфига Redis: %w", err)
	}

	redisClient := infrastructure.NewRedisClient(redisCfg)
	rabbitMQClient, err := infrastructure.NewRabbitMQClient(rabbitCfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к RabbitMQ: %w", err)
	}

	// Загружаем конфиг для отправки уведомлений (endpoint из ENV)
	notificationCfg, _ := config.LoadNotificationConfig(envPath)
	endpoint := notificationCfg.Endpoint
	if endpoint == "" {
		endpoint = sender.DefaultNotificationEndpoint
	}

	// Используем HTTPSender с endpoint из ENV (если пусто — дефолт)
	httpSender := sender.NewHTTPSenderWithEndpoint(10*time.Second, endpoint)
	log.Printf("[Server] Используется HTTPSender для отправки на %s", endpoint)

	notificationService, err := service.NewNotificationService(
		redisClient,
		rabbitMQClient,
		httpSender,
		"delayed_queue",
		"ready_queue",
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервиса: %w", err)
	}

	return notificationService, nil
}

func NewServer(addr string, notificationSvc *service.NotificationService) *Server {
	handler := handlers.NewDefaultHandler(notificationSvc)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Route("/tasks", func(r chi.Router) {
		r.Post("/", handler.CreateTaskHandler)                 // POST /tasks
		r.Get("/", handler.ListTasksHandler)                   // GET /tasks?user_id=...&status=...&category=...
		r.Get("/{task_id}", handler.GetTaskStatusHandler)      // GET /tasks/{task_id}
		r.Post("/{task_id}/cancel", handler.CancelTaskHandler) // POST /tasks/{task_id}/cancel
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

func (s *Server) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("[Server] Запуск Delayed Worker...")
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Server] Delayed Worker упал: %v", r)
			}
		}()
		if err := s.notificationSvc.StartDelayedWorker(ctx); err != nil && err != context.Canceled {
			log.Printf("[Server] Ошибка delayed воркера: %v", err)
		}
	}()

	go func() {
		log.Println("[Server] Запуск Ready Worker...")
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Server] Ready Worker упал: %v", r)
			}
		}()
		if err := s.notificationSvc.StartReadyWorker(ctx); err != nil && err != context.Canceled {
			log.Printf("[Server] Ошибка ready воркера: %v", err)
		}
	}()

	go func() {
		log.Printf("[Server] HTTP сервер запущен на %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[Server] Ошибка запуска сервера: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Server] Завершение работы сервера...")
	cancel() // отменяем контекст для воркеров

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[Server] Ошибка при остановке сервера: %v", err)
	}

	log.Println("[Server] Сервер остановлен корректно")
}
