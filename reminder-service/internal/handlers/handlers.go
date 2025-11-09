package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"L3.1/internal/models"
	"L3.1/internal/service"
)

// Handlers — интерфейс для хендлеров (если ты реально планируешь мокать их в тестах)
type Handlers interface {
	CreateNotificationHandler(w http.ResponseWriter, r *http.Request)
	GetStatusHandler(w http.ResponseWriter, r *http.Request)
	CancelNotificationHandler(w http.ResponseWriter, r *http.Request)
}

// DefaultHandler — основная реализация HTTP-хендлеров
type DefaultHandler struct {
	svc *service.NotificationService
}

// NewDefaultHandler — конструктор, чтобы было удобно создавать хендлер
func NewDefaultHandler(svc *service.NotificationService) *DefaultHandler {
	return &DefaultHandler{svc: svc}
}

// CreateNotificationHandler — POST /notify
func (dh *DefaultHandler) CreateNotificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ошибка при разборе JSON", http.StatusBadRequest)
		return
	}

	id, err := dh.svc.CreateNotification(r.Context(), req.To, req.Subject, req.Body, req.SendAt)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка при создании уведомления: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"id":"%s"}`, id)
}

// GetStatusHandler — GET /notify/status?id=...
func (dh *DefaultHandler) GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "параметр id обязателен", http.StatusBadRequest)
		return
	}

	status, err := dh.svc.GetStatus(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка при получении статуса: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// CancelNotificationHandler — DELETE /notify/cancel?id=...
func (dh *DefaultHandler) CancelNotificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "параметр id обязателен", http.StatusBadRequest)
		return
	}

	if err := dh.svc.CancelNotification(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("ошибка при отмене уведомления: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"canceled"}`)
}
