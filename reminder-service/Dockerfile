# Build stage
FROM golang:1.22-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o reminder-service ./cmd/app/main.go

# Final stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/reminder-service ./reminder-service

EXPOSE 8080

ENTRYPOINT ["./reminder-service"]

