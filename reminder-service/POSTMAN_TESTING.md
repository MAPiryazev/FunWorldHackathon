# Тестирование Reminder Service через Postman

## Базовый URL
```
 http://localhost:8907
```

---

## 1. POST /tasks — Создание задачи

**Метод:** `POST`  
**URL:** `http://localhost:8907/tasks`  
**Headers:**
```
Content-Type: application/json
```

**Body (raw JSON):**
```json
{
  "user_id": "user123",
  "text": "Встреча с командой в 15:00",
  "remind_at": "2025-01-20T15:00:00Z",
  "complexity": 3,
  "priority": "high",
  "category": "работа",
  "notes": "Важная встреча, не опаздывать"
}
```

**Минимальный запрос (только обязательные поля):**
```json
{
  "user_id": "user123",
  "text": "Купить молоко",
  "remind_at": "2025-01-20T18:00:00Z",
  "complexity": 1
}
```

**Успешный ответ (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user123",
  "text": "Встреча с командой в 15:00",
  "remind_at": "2025-01-20T15:00:00Z",
  "status": "pending",
  "retry_count": 0,
  "created_at": "2025-01-20T10:00:00Z",
  "updated_at": "2025-01-20T10:00:00Z",
  "complexity": 3,
  "priority": "high",
  "category": "работа",
  "notes": "Важная встреча, не опаздывать"
}
```

**Важно:** Сохрани `id` из ответа — он понадобится для других запросов!

---

## 2. GET /tasks/{task_id} — Получить статус задачи

**Метод:** `GET`  
**URL:** `http://localhost:8907/tasks/{task_id}`

**Пример:**
```
http://localhost:8907/tasks/550e8400-e29b-41d4-a716-446655440000
```

**Успешный ответ (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user123",
  "text": "Встреча с командой в 15:00",
  "remind_at": "2025-01-20T15:00:00Z",
  "status": "pending",
  "retry_count": 0,
  "created_at": "2025-01-20T10:00:00Z",
  "updated_at": "2025-01-20T10:00:00Z",
  "error": "",
  "complexity": 3,
  "priority": "high",
  "category": "работа",
  "notes": "Важная встреча, не опаздывать"
}
```

**Возможные статусы:**
- `pending` — задача создана, ожидает времени отправки
- `ready` — время наступило, задача в очереди на отправку
- `sent` — задача успешно отправлена
- `failed` — ошибка при отправке (после всех попыток)
- `cancelled` — задача отменена

**Ошибка (404 Not Found):**
```
задача с ID 550e8400-e29b-41d4-a716-446655440000 не найдена
```

---

## 3. POST /tasks/{task_id}/cancel — Отменить задачу

**Метод:** `POST`  
**URL:** `http://localhost:8907/tasks/{task_id}/cancel`

**Пример:**
```
http://localhost:8907/tasks/550e8400-e29b-41d4-a716-446655440000/cancel
```

**Body:** не требуется (можно оставить пустым)

**Успешный ответ (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user123",
  "text": "Встреча с командой в 15:00",
  "remind_at": "2025-01-20T15:00:00Z",
  "status": "cancelled",
  "retry_count": 0,
  "created_at": "2025-01-20T10:00:00Z",
  "updated_at": "2025-01-20T10:30:00Z",
  "error": "",
  "complexity": 3,
  "priority": "high",
  "category": "работа",
  "notes": "Важная встреча, не опаздывать"
}
```

**Ошибка (404 Not Found):**
```
задача с ID 550e8400-e29b-41d4-a716-446655440000 не найдена
```

**Ошибка (если задача уже отправлена):**
```
задачу с ID ... нельзя отменить, статус: sent
```

---

## 4. GET /tasks — Список задач пользователя

**Метод:** `GET`  
**URL:** `http://localhost:8907/tasks?user_id={user_id}&status={status}&category={category}`

**Обязательный параметр:**
- `user_id` — ID пользователя

**Опциональные параметры:**
- `status` — фильтр по статусу (`pending`, `ready`, `sent`, `failed`, `cancelled`)
- `category` — фильтр по категории

**Примеры:**

1. **Все задачи пользователя:**
```
http://localhost:8907/tasks?user_id=user123
```

2. **Только pending задачи:**
```
http://localhost:8907/tasks?user_id=user123&status=pending
```

3. **Задачи категории "работа":**
```
http://localhost:8907/tasks?user_id=user123&category=работа
```

4. **Комбинированный фильтр:**
```
http://localhost:8907/tasks?user_id=user123&status=sent&category=работа
```

**Успешный ответ (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "user123",
    "text": "Встреча с командой в 15:00",
    "remind_at": "2025-01-20T15:00:00Z",
    "status": "pending",
    "retry_count": 0,
    "created_at": "2025-01-20T10:00:00Z",
    "updated_at": "2025-01-20T10:00:00Z",
    "error": "",
    "complexity": 3,
    "priority": "high",
    "category": "работа",
    "notes": "Важная встреча, не опаздывать"
  },
  {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "user_id": "user123",
    "text": "Купить молоко",
    "remind_at": "2025-01-20T18:00:00Z",
    "status": "sent",
    "retry_count": 0,
    "created_at": "2025-01-20T09:00:00Z",
    "updated_at": "2025-01-20T18:00:05Z",
    "error": "",
    "complexity": 1,
    "priority": "",
    "category": "",
    "notes": ""
  }
]
```

**Пустой список (если нет задач):**
```json
[]
```

**Ошибка (400 Bad Request):**
```
параметр user_id обязателен
```

---

## Сценарии тестирования

### Сценарий 1: Создание и проверка задачи
1. **POST /tasks** — создай задачу на ближайшее время (например, через 1 минуту)
2. **GET /tasks/{task_id}** — проверь, что статус `pending`
3. Подожди 1-2 минуты
4. **GET /tasks/{task_id}** — проверь, что статус изменился на `ready` или `sent`

### Сценарий 2: Отмена задачи
1. **POST /tasks** — создай задачу
2. **GET /tasks/{task_id}** — убедись, что статус `pending`
3. **POST /tasks/{task_id}/cancel** — отмени задачу
4. **GET /tasks/{task_id}** — проверь, что статус `cancelled`

### Сценарий 3: Список задач
1. **POST /tasks** — создай несколько задач с разными `user_id`, `status`, `category`
2. **GET /tasks?user_id=user123** — получи все задачи пользователя
3. **GET /tasks?user_id=user123&status=pending** — отфильтруй по статусу
4. **GET /tasks?user_id=user123&category=работа** — отфильтруй по категории

### Сценарий 4: Ошибки
1. **POST /tasks** с невалидными данными (без `user_id` или `text`) — должен вернуть 400
2. **GET /tasks/{несуществующий_id}** — должен вернуть 404
3. **POST /tasks/{несуществующий_id}/cancel** — должен вернуть 404
4. **GET /tasks** без `user_id` — должен вернуть 400

---

## Формат времени (remind_at)

Используй формат **RFC3339** (ISO 8601):
```
2025-01-20T15:00:00Z        // UTC время
2025-01-20T15:00:00+03:00   // с часовым поясом
```

**Примеры:**
- Сейчас + 5 минут: `2025-01-20T15:05:00Z`
- Завтра в 10:00: `2025-01-21T10:00:00Z`
- Через час: `2025-01-20T16:00:00Z`

---

## Проверка отправки уведомлений

Когда задача готова к отправке, сервис отправляет POST запрос на:
```
http://localhost:8081/notifications
```

Чтобы протестировать это, можно:
1. Запустить простой HTTP сервер на порту 8081 (например, через `nc` или Python)
2. Или использовать сервис типа [webhook.site](https://webhook.site) и изменить endpoint в коде

**Формат отправляемого JSON:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user123",
  "text": "Встреча с командой в 15:00",
  "remind_at": "2025-01-20T15:00:00Z",
  "status": "ready",
  "retry_count": 0,
  "created_at": "2025-01-20T10:00:00Z",
  "updated_at": "2025-01-20T15:00:00Z",
  "complexity": 3,
  "priority": "high",
  "category": "работа",
  "notes": "Важная встреча, не опаздывать"
}
```

---

## Быстрый старт в Postman

1. Создай новую коллекцию "Reminder Service"
2. Создай переменную окружения:
   - `base_url` = `http://localhost:8907`
3. Используй переменную в запросах: `{{base_url}}/tasks`
4. Сохраняй `task_id` из ответов в переменные для последующих запросов

