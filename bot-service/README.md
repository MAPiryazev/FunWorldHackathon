# MAX Bot Service

MVP сервис бота для мессенджера **MAX**, интегрированный с NLP- и reminder-сервисами. Приложение принимает webhook-запросы от MAX, передаёт текст в NLP для анализа, создаёт напоминания через reminder-service и отправляет пользователю ответы. Для локальной разработки используется туннель через [Tuna](https://my.tuna.am/).

## Возможности

- **Интеграция с MAX Messenger** - прием и отправка сообщений через webhook
- **Обработка команд** - `/start`, `/help`, `/tasks`, `/cancel`
- **Создание напоминаний** - парсинг естественного языка через NLP-сервис
- **Управление задачами** - просмотр списка задач пользователя
- **Автоматический привет** - автоматическая команда `/start` при первом входе
- **Туннелирование** - автоматический запуск Tuna для локальной разработки

## Требования

- Python 3.11+
- MAX Bot Token 
- Tuna CLI (для туннелирования)
- Доступ к NLP Service (по умолчанию `http://localhost:8000`)
- Доступ к Reminder Service (по умолчанию `http://localhost:8907`)

## Установка

1. **Установите зависимости:**
   ```bash
   pip install -r req.txt
   ```

2. **Установите Tuna CLI:**
   ```bash
   # Следуйте инструкциям на https://tuna.am/docs/getting-started/
   tuna login
   ```

3. **Создайте файл `.env` в корне `bot-service/`:**
   ```env
   # MAX Messenger
   MAX_BOT_TOKEN=your_max_bot_token
   MAX_WEBHOOK_SECRET=optional_shared_secret

   # Интеграции
   NLP_SERVICE_URL=http://localhost:8000
   REMINDER_SERVICE_URL=http://localhost:8907

   # Веб-приложение
   BOT_HOST=0.0.0.0
   BOT_PORT=8080
   WEBHOOK_PATH=/max/webhook
   DEBUG=false

   # Tuna (опционально)
   TUNA_AUTOSTART=true
   TUNA_TOKEN=your_cli_token
   TUNA_LOCATION=ru
   # TUNA_SUBDOMAIN=your-friendly-name
   # TUNA_PUBLIC_URL=https://your-custom-url.tuna.am  # если хотите указать вручную
   ```

## Запуск

### Локальная разработка

```bash
uvicorn app.main:app --host 0.0.0.0 --port 8080 --reload
```

Приложение автоматически:
- Запустит Tuna туннель (если `TUNA_AUTOSTART=true`)
- Установит webhook в MAX на публичный URL туннеля
- Начнет принимать сообщения от пользователей

### Ручной запуск туннеля

Если требуется управлять туннелем вручную:

```bash
tuna http 8080
```

Скопируйте выданный HTTPS-URL и пропишите его в `.env` в `TUNA_PUBLIC_URL`. После перезапуска приложения webhook будет указывать на этот адрес.

### Docker

```bash
docker compose up bot-service
```


## API Endpoints

### `GET /health`
Проверка состояния сервиса. Возвращает:
```json
{
  "status": "ok",
  "webhook": "https://your-tunnel.tuna.am/max/webhook"
}
```

### `POST /max/webhook`
Webhook для приема сообщений от MAX Messenger. Обрабатывается автоматически.

### `POST /notifications`
Прием уведомлений от reminder-service о наступлении времени напоминания.

**Request:**
```json
{
  "user_id": "50789519",
  "chat_id": "50789519",
  "id": "task-uuid",
  "text": "Позвонить врачу",
  "remind_at": "2025-11-13T21:12:00Z",
  "priority": "medium",
  "category": "здоровье",
  "complexity": 1,
  "notes": ""
}
```

## Команды бота

- `/start` - начать работу с ботом (автоматически выполняется при первом входе)
- `/help` - показать справку по командам
- `/tasks` - просмотреть список всех задач пользователя
- `/cancel` - отменить текущее действие

