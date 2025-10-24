package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var RedisClient *redis.Client

// InitRedis инициализирует клиент Redis
func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("DEBUG: Redis ping error: %v\n", err)
	} else {
		fmt.Println("DEBUG: Redis connected OK")
	}
}

// SetSession сохраняет сессию в Redis
func SetSession(token string, userID int, role string, ttl time.Duration) error {
	key := "session:" + token
	data := map[string]interface{}{
		"user_id": userID,
		"role":    role,
	}
	if err := RedisClient.HSet(ctx, key, data).Err(); err != nil {
		fmt.Printf("DEBUG: Redis HSet error: %v\n", err)
		return err
	}
	if err := RedisClient.Expire(ctx, key, ttl).Err(); err != nil {
		fmt.Printf("DEBUG: Redis Expire error: %v\n", err)
		return err
	}
	fmt.Printf("DEBUG: Redis SetSession OK key=%s ttl=%v\n", key, ttl)
	return nil
}

// GetSession достаёт сессию (для отладки)
func GetSession(token string) (map[string]string, error) {
	key := "session:" + token
	res, err := RedisClient.HGetAll(ctx, key).Result()
	if err != nil {
		fmt.Printf("DEBUG: Redis GetSession error: %v\n", err)
		return nil, err
	}
	return res, nil
}

// DeleteSession удаляет токен из Redis
func DeleteSession(token string) error {
	if RedisClient == nil {
		return nil
	}
	return RedisClient.Del(ctx, token).Err()
}
