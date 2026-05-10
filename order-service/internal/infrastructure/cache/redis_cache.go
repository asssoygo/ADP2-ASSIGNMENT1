package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"order-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (r *RedisCache) Get(ctx context.Context, key string) (*domain.Order, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, errors.New("cache miss")
		}
		return nil, err
	}
	var order domain.Order
	if err := json.Unmarshal([]byte(val), &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, order *domain.Order, ttl time.Duration) error {
	data, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
