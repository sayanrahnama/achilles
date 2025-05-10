package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value any, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

type redisClusterRepository struct {
	client *redis.ClusterClient
}

func NewRedisClusterRepository(client *redis.ClusterClient) RedisRepository {
	return &redisClusterRepository{
		client: client,
	}
}
func (r *redisClusterRepository) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *redisClusterRepository) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *redisClusterRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
