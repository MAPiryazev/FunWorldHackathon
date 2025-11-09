# FunWorldHackathon
Effectiveness calendar bot for Max application

# **Примерная** архитектура по папкам
/max-reminder-bot/
│
├── bot-service/                # Python бот (MAX API)
│   ├── app/
│   │   ├── main.py             # Точка входа
│   │   ├── handlers/           # Обработчики команд
│   │   ├── models/             # Pydantic схемы / dataclasses
│   │   ├── services/
│   │   │   ├── max_api.py      # Работа с MAX API
│   │   │   ├── queue_client.py # Отправка задач в RabbitMQ
│   │   │   ├── nlp_client.py   # HTTP-клиент к nlp-service
│   │   ├── utils/
│   │   └── config.py           # Настройки (env)
│   ├── Dockerfile
│   └── requirements.txt
│
├── nlp-service/                # Python микросервис (нейронка)
│   ├── app/
│   │   ├── main.py             # FastAPI endpoint
│   │   ├── parser.py           # логика разбора текста
│   │   ├── models.py
│   │   ├── config.py
│   ├── Dockerfile
│   └── requirements.txt
│
├── reminder-service/           # Go-сервис с Redis + RabbitMQ
│   ├── cmd/
│   │   └── reminder-service/
│   │       └── main.go
│   ├── internal/
│   │   ├── queue/              # RabbitMQ consumer
│   │   ├── scheduler/          # логика планирования
│   │   ├── storage/            # Redis-хранилище
│   │   ├── models/
│   │   └── api/                # (если есть REST API)
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── docker-compose.yml          # собирает всё вместе
├── .env                        # общие переменные окружения
└── README.md
