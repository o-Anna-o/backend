package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Repository — централизованный репозиторий с DB, Redis и конфигом JWT.
type Repository struct {
	db          *gorm.DB
	redisClient *redis.Client
	jwtKey      string
}

// New — инициализация репозитория.
// postgresDSN — строка подключения к Postgres (DSN)
// redisAddr — "host:port", redisPass — пароль (может быть "")
// jwtKey — секрет для подписи JWT
func New(postgresDSN, redisAddr, redisPass, jwtKey string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(postgresDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})

	// проверка соединения с Redis
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		// не фаталим — но возвращаем ошибку, чтобы ты знала
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	repo := &Repository{
		db:          db,
		redisClient: rdb,
		jwtKey:      jwtKey,
	}
	return repo, nil
}

// Redis возвращает клиент Redis
func (r *Repository) Redis() *redis.Client {
	return r.redisClient
}

// DB возвращает *gorm.DB (может быть полезно)
func (r *Repository) DB() *gorm.DB {
	return r.db
}

// JWTKey возвращает секретный ключ для подписи токенов
func (r *Repository) JWTKey() string {
	return r.jwtKey
}
