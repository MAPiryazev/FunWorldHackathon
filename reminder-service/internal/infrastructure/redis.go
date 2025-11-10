package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"reminder-service/internal/models"

	"github.com/redis/go-redis/v9"
)

// RedisClient — клиент для взаимодействия с Redis.
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient — конструктор клиента Redis.
func NewRedisClient(addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisClient{client: rdb}
}

// keyForReminder — формирует ключ для хранения напоминания в Redis.
func keyForReminder(id string) string {
	return fmt.Sprintf("reminder:%s", id)
}

// Save — сохраняет напоминание без TTL.
func (rc *RedisClient) Save(ctx context.Context, message *models.RedisMessage) error {
	if message == nil {
		return fmt.Errorf("нельзя сохранить nil сообщение в redis")
	}

	if len(message.UserID) == 0 {
		return fmt.Errorf("нельзя сохранить сообщение без userID")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("ошибка при json.Marshal сообщения в redis: %w", err)
	}

	key := keyForReminder(message.ID)
	if err := rc.client.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("не удалось сохранить сообщение в redis: %w", err)
	}

	return nil
}

// SaveWithTTL — сохраняет напоминание с TTL (для автоудаления Redis’ом).
func (rc *RedisClient) SaveWithTTL(ctx context.Context, message *models.RedisMessage, ttl time.Duration) error {
	if message == nil {
		return fmt.Errorf("нельзя сохранить nil сообщение в redis")
	}

	if len(message.UserID) == 0 {
		return fmt.Errorf("нельзя сохранить сообщение без userID")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("ошибка при json.Marshal сообщения в redis: %w", err)
	}

	key := keyForReminder(message.ID)
	if err := rc.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("не удалось сохранить сообщение в redis: %w", err)
	}

	return nil
}

// Get — получает напоминание из Redis по его ID.
func (rc *RedisClient) Get(ctx context.Context, id string) (*models.RedisMessage, error) {
	key := keyForReminder(id)

	data, err := rc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("напоминание с id %s не найдено", id)
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении напоминания %s из Redis: %w", id, err)
	}

	var msg models.RedisMessage
	if err := json.Unmarshal([]byte(data), &msg); err != nil {
		return nil, fmt.Errorf("ошибка при разборе данных напоминания %s: %w", id, err)
	}

	return &msg, nil
}

// Exists — проверяет, существует ли напоминание в Redis.
func (rc *RedisClient) Exists(ctx context.Context, id string) (bool, error) {
	key := keyForReminder(id)

	exists, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("ошибка при проверке существования напоминания %s: %w", id, err)
	}

	return exists > 0, nil
}

// Delete — удаляет напоминание из Redis по его ID.
func (rc *RedisClient) Delete(ctx context.Context, id string) error {
	key := keyForReminder(id)

	if err := rc.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("ошибка при удалении напоминания %s из Redis: %w", id, err)
	}

	return nil
}

// ListByUser — возвращает все напоминания пользователя по userID.
func (rc *RedisClient) ListByUser(ctx context.Context, userID string) ([]*models.RedisMessage, error) {
	if len(userID) == 0 {
		return nil, fmt.Errorf("userID не может быть пустым")
	}

	var results []*models.RedisMessage
	var cursor uint64

	for {
		// сканируем порциями, чтобы не блокировать Redis
		keys, newCursor, err := rc.client.Scan(ctx, cursor, "reminder:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("ошибка при сканировании ключей Redis: %w", err)
		}

		for _, key := range keys {
			data, err := rc.client.Get(ctx, key).Result()
			if err != nil {
				if err == redis.Nil {
					continue // ключ уже удалён TTL
				}
				return nil, fmt.Errorf("ошибка при чтении ключа %s: %w", key, err)
			}

			var msg models.RedisMessage
			if err := json.Unmarshal([]byte(data), &msg); err != nil {
				continue // пропускаем битые записи
			}

			// фильтруем по userID
			if msg.UserID == userID {
				results = append(results, &msg)
			}
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return results, nil
}
