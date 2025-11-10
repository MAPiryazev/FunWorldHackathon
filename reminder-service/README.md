# Reminder Service

Сервис отложенной отправки уведомлений. Принимает задачи через HTTP API, планирует их отправку на указанное время и отправляет через внешний endpoint.

## Архитектура

1. **HTTP API** — принимает запросы на создание/управление задачами
2. **Delayed Queue** (RabbitMQ) — очередь задач, которые нужно отправить позже
3. **Delayed Worker** — проверяет задачи и переносит готовые в `ready_queue`
4. **Ready Queue** (RabbitMQ) — очередь задач, готовых к отправке
5. **Ready Worker** — отправляет задачи через HTTP на внешний endpoint
6. **Redis** — хранит задачи для быстрого доступа

## API Endpoints

### POST /tasks
Создаёт новую задачу/напоминание.

**Request Body:**
```json
{
  "user_id": "user123",
  "text": "Встреча в 15:00",
  "remind_at": "2025-01-15T15:00:00Z",
  "complexity": 3,
  "priority": "high",
  "category": "работа",
  "notes": "Важная встреча"
}
```

**Поля:**
- `user_id` (обязательно) — ID пользователя
- `text` (обязательно) — текст уведомления
- `remind_at` (обязательно) — время отправки (RFC3339)
- `complexity` (обязательно) — сложность 1-5
- `priority` (опционально) — приоритет: `low`, `medium`, `high`
- `category` (опционально) — категория
- `notes` (опционально) — заметки

**Response:** `201 Created` с созданной задачей (включая `id`)

---

### GET /tasks/{task_id}
Возвращает статус задачу по ID.

**Response:**
```json
{
  "id": "uuid",
  "user_id": "user123",
  "text": "Встреча в 15:00",
  "remind_at": "2025-01-15T15:00:00Z",
  "status": "pending",
  "retry_count": 0,
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T10:00:00Z",
  "complexity": 3,
  "priority": "high",
  "category": "работа",
  "notes": "Важная встреча"
}
```

---

### POST /tasks/{task_id}/cancel
Отменяет задачу. Можно отменить только задачи со статусом `pending` или `ready`.

**Response:** `200 OK` с обновлённой задачей (статус `cancelled`)

---

### GET /tasks?user_id={user_id}&status={status}&category={category}
Возвращает список задач пользователя с фильтрацией.

**Query параметры:**
- `user_id` (обязательно) — ID пользователя
- `status` (опционально) — фильтр по статусу
- `category` (опционально) — фильтр по категории

**Response:** `200 OK` с массивом задач

---

## Как работает с сообщениями

### Жизненный цикл задачи

1. **Создание** (`POST /tasks`)
   - Задача сохраняется в Redis со статусом `pending`
   - Задача публикуется в `delayed_queue` (RabbitMQ)

2. **Ожидание** (Delayed Worker)
   - Worker проверяет задачи из `delayed_queue`
   - Если `remind_at` наступило → переносит в `ready_queue` со статусом `ready`
   - Если ещё рано → возвращает обратно в `delayed_queue`

3. **Отправка** (Ready Worker)
   - Worker берёт задачу из `ready_queue`
   - Отправляет HTTP POST на внешний endpoint: `http://localhost:8081/notifications`
   - При успехе → статус `sent`
   - При ошибке → статус `failed`, повтор через экспоненциальную задержку (до 5 попыток)

4. **Отмена** (`POST /tasks/{id}/cancel`)
   - Статус меняется на `cancelled`
   - Задача не будет отправлена, даже если уже в очереди

### Формат отправки на внешний endpoint

Когда задача готова к отправке, сервис отправляет POST запрос на `http://localhost:8081/notifications`:

```json
{
  "id": "uuid",
  "user_id": "user123",
  "text": "Встреча в 15:00",
  "remind_at": "2025-01-15T15:00:00Z",
  "status": "ready",
  "retry_count": 0,
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T10:00:00Z",
  "complexity": 3,
  "priority": "high",
  "category": "работа",
  "notes": "Важная встреча"
}
```

---

## Запуск

### Локально
```bash
go run cmd/app/main.go
```

### Docker
```bash
docker-compose up --build -d
```

Сервис доступен на `http://localhost:8907`

---

## Конфигурация

Переменные окружения (`.env`):
- `RABBITMQ_HOST`, `RABBITMQ_PORT`, `RABBITMQ_DEFAULT_USER`, `RABBITMQ_DEFAULT_PASS`
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- `API_HOST`, `API_PORT`

Endpoint для отправки уведомлений фиксирован в коде: `http://localhost:8081/notifications` (можно изменить в `internal/sender/sender.go`)
