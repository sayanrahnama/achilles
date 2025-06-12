package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID, refreshToken string, expiration time.Duration) error
	GetRefreshToken(ctx context.Context, userID string) (string, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
}

type tokenRepositoryImpl struct {
	RDB *redis.ClusterClient
}

func NewTokenRepository(rdb *redis.ClusterClient) TokenRepository {
	return &tokenRepositoryImpl{
		RDB: rdb,
	}
}

func (r *tokenRepositoryImpl) StoreRefreshToken(ctx context.Context, userID, refreshToken string, expiration time.Duration) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	return r.RDB.Set(ctx, key, refreshToken, expiration).Err()
}

func (r *tokenRepositoryImpl) GetRefreshToken(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("refresh_token:%s", userID)
	result, err := r.RDB.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil
		}
		return "", err
	}
	return result, nil
}

func (r *tokenRepositoryImpl) DeleteRefreshToken(ctx context.Context, userID string) error {
	key := fmt.Sprintf("refresh_token:%s", userID)
	return r.RDB.Del(ctx, key).Err()
}
