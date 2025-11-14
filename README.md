# FunWorldHackathon

Эффективный чат-бот для мессенджера MAX. Система позволяет пользователям создавать напоминания на естественном языке, которые автоматически парсятся, планируются и отправляются в указанное время.

##  Архитектура

Проект представляет собой микросервисную архитектуру, состоящую из трех основных сервисов:

1. **bot-service** (Python/FastAPI) - основной бот-сервис, интегрированный с MAX Messenger
2. **nlp-service** (Python/FastAPI) - сервис парсинга текстовых напоминаний с использованием GigaChat API
3. **reminder-service** (Go) - сервис планирования и отправки напоминаний через RabbitMQ и Redis

##  Установка и запуск

### Требования

- Docker, установленный ПК
- MAX Bot Token
- GigaChat API ключ
- Tuna CLI (для локальной разработки bot-service)

### Запуск через Docker Compose

**Клонируйте репозиторий:**
   ```bash
   git clone <https://github.com/MAPiryazev/FunWorldHackathon.git>
   cd FunWorldHackathon
   ```

**Запустите все сервисы:**
   ```bash
   docker compose up --build
   ```

## Команды бота

- `/start` - начать работу с ботом (автоматически выполняется при первом входе)
- `/help` - показать справку по командам
- `/tasks` - просмотреть список всех задач пользователя
- `/cancel` - отменить текущее действие


##  Сервисы

### bot-service

Основной сервис бота для мессенджера MAX. Принимает webhook-запросы от MAX, передаёт текст в NLP для анализа, создаёт напоминания через reminder-service и отправляет пользователю ответы.

**Основные возможности:**
- Интеграция с MAX Messenger через webhook
- Обработка команд: `/start`, `/help`, `/tasks`, `/cancel`
- Создание напоминаний через парсинг естественного языка
- Управление задачами пользователя
- Автоматический запуск Tuna туннеля для локальной разработки

**API Endpoints:**
- `GET /health` - проверка состояния сервиса
- `POST /max/webhook` - webhook для приема сообщений от MAX
- `POST /notifications` - прием уведомлений от reminder-service

Подробнее: [bot-service/README.md](bot-service/README.md)

### nlp-service

Микросервис для парсинга текстовых напоминаний с использованием GigaChat API. Извлекает структурированные данные из неформатированного текста на естественном языке.

**Основные возможности:**
- Парсинг текстовых напоминаний на естественном языке
- Извлечение даты, времени, приоритета, категории, сложности
- Интеграция с GigaChat API
- Валидация и корректировка извлеченных данных
- Автоматическое определение времени с учетом московского времени

**API Endpoints:**
- `POST /parse` - парсинг текста напоминания

**Request:**
```json
{
  "text": "Напомни завтра в 10:00 позвонить маме",
  "user_id": "50789519"
}
```

**Response:**
```json
{
  "user_id": "50789519",
  "text": "позвонить маме",
  "remind_at": "2025-11-14T10:00:00+03:00",
  "complexity": 1,
  "priority": "medium",
  "category": "личное",
  "notes": null
}
```

Подробнее: [nlp-service/README.md](nlp-service/README.md)

### reminder-service

Сервис отложенной отправки уведомлений. Принимает задачи через HTTP API, планирует их отправку на указанное время и отправляет через внешний endpoint.

**Архитектура:**
1. **HTTP API** - принимает запросы на создание/управление задачами
2. **Delayed Queue** (RabbitMQ) - очередь задач, которые нужно отправить позже
3. **Delayed Worker** - проверяет задачи и переносит готовые в `ready_queue`
4. **Ready Queue** (RabbitMQ) - очередь задач, готовых к отправке
5. **Ready Worker** - отправляет задачи через HTTP на внешний endpoint
6. **Redis** - кэш задач для быстрого доступа

**API Endpoints:**
- `POST /tasks` - создание новой задачи/напоминания
- `GET /tasks/{task_id}` - получение задачи по ID
- `POST /tasks/{task_id}/cancel` - отмена задачи
- `GET /tasks?user_id={user_id}&status={status}&category={category}` - список задач с фильтрацией

**Жизненный цикл задачи:**
1. **Создание** (`POST /tasks`) - задача сохраняется в Redis со статусом `pending` и публикуется в `delayed_queue`
2. **Ожидание** (Delayed Worker) - проверяет задачи, если время наступило → переносит в `ready_queue` со статусом `ready`
3. **Отправка** (Ready Worker) - отправляет HTTP POST на `http://bot-service:8080/notifications`
4. **Отмена** (`POST /tasks/{id}/cancel`) - статус меняется на `cancelled`

Подробнее: [reminder-service/README.md](reminder-service/README.md)

## Конфигурация

### Переменные окружения

#### Общие (.env в корне)
- `TZ` - временная зона (по умолчанию `Europe/Moscow`)

#### bot-service
- `MAX_BOT_TOKEN` - токен бота MAX (обязательно)
- `MAX_WEBHOOK_SECRET` - секретный ключ для webhook (опционально)
- `NLP_SERVICE_URL` - URL NLP сервиса (по умолчанию `http://localhost:8000`)
- `REMINDER_SERVICE_URL` - URL Reminder сервиса (по умолчанию `http://localhost:8907`)
- `BOT_HOST` - хост для бота (по умолчанию `0.0.0.0`)
- `BOT_PORT` - порт для бота (по умолчанию `8080`)
- `WEBHOOK_PATH` - путь для webhook (по умолчанию `/max/webhook`)
- `DEBUG` - режим отладки (по умолчанию `false`)
- `TUNA_AUTOSTART` - автоматический запуск Tuna (по умолчанию `true`)
- `TUNA_TOKEN` - токен Tuna для туннелирования
- `TUNA_LOCATION` - локация Tuna (например, `ru`)

#### nlp-service
- `GIGACHAT_AUTH_KEY` - ключ авторизации GigaChat (обязательно)
- `GIGACHAT_SCOPE` - область доступа GigaChat (по умолчанию `GIGACHAT_API_PERS`)
- `APP_HOST` - хост приложения (по умолчанию `0.0.0.0`)
- `APP_PORT` - порт приложения (по умолчанию `8000`)
- `DEBUG` - режим отладки (по умолчанию `false`)

#### reminder-service
- `RABBITMQ_HOST` - хост RabbitMQ (по умолчанию `reminder-rabbitmq`)
- `RABBITMQ_PORT` - порт RabbitMQ (по умолчанию `5672`)
- `REDIS_HOST` - хост Redis (по умолчанию `reminder-redis`)
- `REDIS_PORT` - порт Redis (по умолчанию `6379`)
- `NOTIFICATION_ENDPOINT` - endpoint для отправки уведомлений (по умолчанию `http://bot-service:8080/notifications`)
- `API_SENDER_ENDPOINT` - endpoint для отправки (по умолчанию `http://bot-service:8080/notifications`)


