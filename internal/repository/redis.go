package repository

import (
	"context"
	"time"
	"fmt"

	"github.com/redis/go-redis/v9"

)

type CacheManager struct {
	client *redis.Client
}

func NewCacheManager() (*CacheManager, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheManager{
		client: rdb,
	}, nil
}

func (cm *CacheManager) SetEmailIndex(ctx context.Context, email string, id string) error {
	key := fmt.Sprintf("email:%s", email)

	err := cm.client.Set(ctx, key, id, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set email index in redis: %w", err)
	}

	return nil
}

func (cm *CacheManager) GetIDByEmail(ctx context.Context, email string) (string, error) {
	key := fmt.Sprintf("email:%s", email)

	id, err := cm.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil 
		}
		return "", fmt.Errorf("failed to get email index from redis: %w", err)
	}

	return id, nil
}
