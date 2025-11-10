package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"reminder-service/internal/models"
	"reminder-service/internal/service"

	"github.com/go-chi/chi/v5"
)

// DefaultHandler — основная реализация HTTP-хендлеров
type DefaultHandler struct {
	svc service.NotificationServiceIface
}

// NewDefaultHandler — конструктор хендлера
func NewDefaultHandler(svc service.NotificationServiceIface) *DefaultHandler {
	return &DefaultHandler{svc: svc}
}

// CreateTaskHandler — POST /tasks
// Создаёт новую задачу/напоминание
func (dh *DefaultHandler) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("ошибка при разборе JSON: %v", err), http.StatusBadRequest)
		return
	}

	task, err := dh.svc.CreateTask(r.Context(), &req)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка при создании задачи: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}
}

// GetTaskStatusHandler — GET /tasks/{task_id}
// Возвращает текущий статус задачи
func (dh *DefaultHandler) GetTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	taskID := chi.URLParam(r, "task_id")
	if taskID == "" {
		http.Error(w, "параметр task_id обязателен", http.StatusBadRequest)
		return
	}

	task, err := dh.svc.GetTaskStatus(r.Context(), taskID)
	if err != nil {
		// Проверяем, это ошибка "не найдено" или другая
		if strings.Contains(err.Error(), "не найдена") || strings.Contains(err.Error(), "не найдено") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("ошибка при получении статуса задачи: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}
}

// CancelTaskHandler — POST /tasks/{task_id}/cancel
// Отменяет задачу
func (dh *DefaultHandler) CancelTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	taskID := chi.URLParam(r, "task_id")
	if taskID == "" {
		http.Error(w, "параметр task_id обязателен", http.StatusBadRequest)
		return
	}

	err := dh.svc.CancelTask(r.Context(), taskID)
	if err != nil {
		// Проверяем, это ошибка "не найдена" или другая
		if strings.Contains(err.Error(), "не найдена") || strings.Contains(err.Error(), "не найдено") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("ошибка при отмене задачи: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем обновлённый статус задачи для ответа
	task, err := dh.svc.GetTaskStatus(r.Context(), taskID)
	if err != nil {
		// Если не удалось получить, всё равно возвращаем успех (задача отменена)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"cancelled"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, "ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}
}

// ListTasksHandler — GET /tasks?user_id={user_id}&status={status}&category={category}
// Возвращает список задач с фильтрацией
func (dh *DefaultHandler) ListTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "параметр user_id обязателен", http.StatusBadRequest)
		return
	}

	// Получаем все задачи пользователя
	tasks, err := dh.svc.ListTasksByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка при получении списка задач: %v", err), http.StatusInternalServerError)
		return
	}

	// Применяем фильтры
	filteredTasks := make([]*models.RedisMessage, 0)
	statusFilter := r.URL.Query().Get("status")
	categoryFilter := r.URL.Query().Get("category")

	for _, task := range tasks {
		// Фильтр по статусу
		if statusFilter != "" && task.Status != statusFilter {
			continue
		}

		// Фильтр по категории
		if categoryFilter != "" && task.Category != categoryFilter {
			continue
		}

		filteredTasks = append(filteredTasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(filteredTasks); err != nil {
		http.Error(w, "ошибка при формировании ответа", http.StatusInternalServerError)
		return
	}
}
