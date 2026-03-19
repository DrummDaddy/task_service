package db

import (
	"context"
	"fmt"

	"github.com/DrummDaddy/task_service/internal/config"
	"github.com/redis/go-redis/v9"
)

func OpenRedis(cfg config.Config) (*redis.Client, error) {
	if cfg.Redis.Addr == "" {
		return nil, fmt.Errorf("Redis Addr is empty ")

	}
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr,
		DB:   cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to connect to Redis with err %v", err)
	}
	return rdb, nil
}
