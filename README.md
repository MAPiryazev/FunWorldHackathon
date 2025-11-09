# FunWorldHackathon
Effectiveness calendar bot for Max application

# **Примерная** архитектура по папкам  
/max-reminder-bot/<br>
│<br>
├── bot-service/                # Python бот (MAX API)<br>
│   ├── app/<br>
│   │   ├── main.py             # Точка входа<br> 
│   │   ├── handlers/           # Обработчики команд<br>
│   │   ├── models/             # Pydantic схемы / dataclasses<br>
│   │   ├── services/<br>
│   │   │   ├── max_api.py      # Работа с MAX API<br>
│   │   │   ├── queue_client.py # Отправка задач в RabbitMQ<br>
│   │   │   ├── nlp_client.py   # HTTP-клиент к nlp-service<br>
│   │   ├── utils/<br>
│   │   └── config.py           # Настройки (env)<br>
│   ├── Dockerfile<br>
│   └── requirements.txt<br>
│<br>
├── nlp-service/                # Python микросервис (нейронка)<br>
│   ├── app/<br>
│   │   ├── main.py             # FastAPI endpoint<br>
│   │   ├── parser.py           # логика разбора текста<br>
│   │   ├── models.py<br>
│   │   ├── config.py<br>
│   ├── Dockerfile<br>
│   └── requirements.txt<br>
│<br>
├── reminder-service/           # Go-сервис с Redis + RabbitMQ<br>
│   ├── cmd/<br>
│   │   └── reminder-service/<br>
│   │       └── main.go<br>
│   ├── internal/<br>
│   │   ├── queue/              # RabbitMQ consumer<br>
│   │   ├── scheduler/          # логика планирования<br>
│   │   ├── storage/            # Redis-хранилище<br>
│   │   ├── models/<br>
│   │   └── api/                # (если есть REST API)<br>
│   ├── go.mod<br>
│   ├── go.sum<br>
│   └── Dockerfile<br>
│<br>
├── docker-compose.yml          # собирает всё вместе<br>
├── .env                        # общие переменные окружения<br>
└── README.md<br>
