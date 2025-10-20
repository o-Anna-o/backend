package repository

import (
	"context"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository struct {
	db     *gorm.DB
	redis  *redis.Client
	jwtKey string // для JWT
}

func New(dsn string, redisEndpoint string, redisPassword string, jwtKey string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisEndpoint,
		Password: redisPassword,
		DB:       0,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &Repository{
		db:     db,
		redis:  rdb,
		jwtKey: jwtKey,
	}, nil
}

func (r *Repository) DB() *gorm.DB {
	return r.db
}

func (r *Repository) Redis() *redis.Client {
	return r.redis
}

func (r *Repository) JWTKey() string {
	return r.jwtKey
}
