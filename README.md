# MAX Bot Service

MVP cервис бота для мессенджера **MAX**, интегрированный с существующими NLP- и reminder-сервисами. Приложение принимает webhook-запросы от MAX, передаёт текст в NLP, создаёт напоминания и отправляет пользователю ответы. Для локальной разработки используется туннель через [Tuna](https://my.tuna.am/).

## Стек

- Python 3.11+
- FastAPI
- maxapi
- Tuna CLI (туннелирование)
- httpx
- Pydantic & pydantic-settings

## Структура

- `app/main.py` — точка входа FastAPI-приложения и webhook для MAX.
- `app/config.py` — загрузка настроек из `.env`.
- `app/handlers/` — существующая бизнес-логика (команды, напоминания, уведомления).
- `app/services/` — клиенты и вспомогательные сервисы (MAX, NLP, Reminder, Tuna).
- `app/models/` — Pydantic-схемы.
- `req.txt` — зависимости проекта.

## Настройка окружения

1. Зарегистрируйте бота в кабинете [MAX](https://dev.max.ru/) и получите токен.
2. Установите [Tuna CLI](https://tuna.am/docs/getting-started/) и выполните `tuna login`, чтобы туннели могли подниматься из кода.
3. Создайте и активируйте виртуальное окружение, затем установите зависимости:
   ```bash
   pip install -r req.txt
   ```
4. Создайте `.env` (значения примерные):
   ```env
   MAX_BOT_TOKEN=your_max_bot_token
   MAX_WEBHOOK_SECRET=optional_shared_secret

   NLP_SERVICE_URL=http://localhost:8000
   REMINDER_SERVICE_URL=http://localhost:8907

   BOT_HOST=0.0.0.0
   BOT_PORT=8081
   DEBUG=true

   # Tuna (опционально)
   TUNA_AUTOSTART=true
   TUNA_TOKEN=your_cli_token
   TUNA_LOCATION=ru
   # TUNA_PUBLIC_URL=https://your-custom-url.tuna.am  # если хотите указать вручную
   ```

## Запуск

```bash
uvicorn app.main:app --host 0.0.0.0 --port 8081 --reload
```

Приложение попробует поднять туннель командой `tuna http 8081` и установить webhook в MAX. Если автозапуск не нужен, установите `TUNA_AUTOSTART=false` и укажите публичный URL через `TUNA_PUBLIC_URL`.

## Ручной запуск туннеля

Если требуется управлять туннелем вручную:

```bash
tuna http 8081
```

Скопируйте выданный HTTPS-URL и пропишите его в `.env` в `TUNA_PUBLIC_URL`. После перезапуска приложения webhook будет указывать на этот адрес.

## Точки API

- `GET /health` — общее состояние, а также текущий webhook-URL (если есть).
- `POST /notifications` — входящие уведомления от reminder-сервиса (логика не менялась).
- `POST /max/webhook` — webhook MAX (обрабатывается автоматически).

## Команды бота

- `/start` — начать работу с ботом
- `/help` — показать справку
- `/cancel` — отменить текущее действие
- `/tasks` — просмотреть список задач
- `/stats` — статистика по задачам

- `/start`, `/help`, `/cancel`, `/tasks`, `/stats` и любая текстовая команда обрабатываются через `MessageHandler`.

## Примечания

- При `DEBUG=true` исходящие сообщения не отправляются в MAX, а только логируются.
- Состояние пользователя по-прежнему хранится в памяти (`StateManager`), для продакшена потребуется внешнее хранилище.
- Убедитесь, что NLP и reminder сервисы доступны по указанным адресам.


