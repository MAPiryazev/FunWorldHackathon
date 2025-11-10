# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Копируем go mod файлы
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o reminder-service ./cmd/app/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник из builder stage
COPY --from=builder /build/reminder-service .

# Копируем .env файл (будет передан через docker-compose)
# Если .env не будет скопирован, приложение попытается использовать переменные окружения

EXPOSE 8080

CMD ["./reminder-service"]

