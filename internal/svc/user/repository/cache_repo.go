package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hailsayan/achilles/internal/svc/user/model"
)

const (
	userCacheTTL = 5 * time.Minute
)

type CacheRepository interface {
	GetUser(ctx context.Context, key string) (*model.User, error)
	SetUser(ctx context.Context, key string, user *model.User) error
	DeleteUser(ctx context.Context, key string) error
	
	UserIDKey(id string) string
	UsernameKey(username string) string
}

type cacheRepository struct {
	redis RedisRepository
}

func NewCacheRepository(redis RedisRepository) CacheRepository {
	return &cacheRepository{
		redis: redis,
	}
}

func (c *cacheRepository) GetUser(ctx context.Context, key string) (*model.User, error) {
	if c.redis == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	
	userJSON, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var user model.User
	if err = json.Unmarshal([]byte(userJSON), &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (c *cacheRepository) SetUser(ctx context.Context, key string, user *model.User) error {
	if c.redis == nil {
		return nil
	}
	
	userBytes, err := json.Marshal(user)
	if err != nil {
		return err
	}
	
	return c.redis.Set(ctx, key, string(userBytes), userCacheTTL)
}

func (c *cacheRepository) DeleteUser(ctx context.Context, key string) error {
	if c.redis == nil {
		return nil
	}
	
	return c.redis.Delete(ctx, key)
}

func (c *cacheRepository) UserIDKey(id string) string {
	return fmt.Sprintf("user:%s", id)
}

func (c *cacheRepository) UsernameKey(username string) string {
	return fmt.Sprintf("username:%s", username)
}