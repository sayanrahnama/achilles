package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hailsayan/achilles/auth/model"
	"github.com/hailsayan/achilles/pkg/logger"
	"github.com/hailsayan/achilles/pkg/redis"
)

const (
	refreshTokenPrefix = "refresh_token:"
	passwordLockPrefix = "password_lock:"
	refreshTokenTTL    = 24 * time.Hour * 7 // 7 days
	passwordLockTTL    = 2 * time.Minute    // Lock during password change
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrOperationLocked   = errors.New("operation is locked")
	ErrTokenNotFound     = errors.New("refresh token not found")
	ErrTokenInvalid      = errors.New("refresh token is invalid")
)

type AuthRepository interface {
	// User Auth Operations
	StoreUserAuth(ctx context.Context, userAuth *model.UserAuth) error
	GetUserAuth(ctx context.Context, userID string) (*model.UserAuth, error)
	UpdateUserPassword(ctx context.Context, userID, hashedPassword string) error
	
	// Token Operations
	StoreRefreshToken(ctx context.Context, userID, refreshToken string) error
	ValidateRefreshToken(ctx context.Context, userID, refreshToken string) (bool, error)
	DeleteRefreshToken(ctx context.Context, userID, refreshToken string) error
	
	// Locking Operations
	AcquirePasswordChangeLock(ctx context.Context, userID string) (bool, error)
	ReleasePasswordChangeLock(ctx context.Context, userID string) error
}

type authRepository struct {
	db     *sql.DB
	redis  redis.Client
	logger logger.Logger
}

func NewAuthRepository(db *sql.DB, redis redis.Client, logger logger.Logger) AuthRepository {
	return &authRepository{
		db:     db,
		redis:  redis,
		logger: logger,
	}
}

func (r *authRepository) StoreUserAuth(ctx context.Context, userAuth *model.UserAuth) error {
	query := `
		INSERT INTO user_auth (id, hashed_password)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET hashed_password = $2
	`

	_, err := r.db.ExecContext(ctx, query, userAuth.ID, userAuth.HashedPassword)
	return err
}

func (r *authRepository) GetUserAuth(ctx context.Context, userID string) (*model.UserAuth, error) {
	query := `
		SELECT id, hashed_password
		FROM user_auth
		WHERE id = $1
	`

	userAuth := &model.UserAuth{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&userAuth.ID,
		&userAuth.HashedPassword,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return userAuth, nil
}

func (r *authRepository) UpdateUserPassword(ctx context.Context, userID, hashedPassword string) error {
	// Check if the password change is locked
	locked, err := r.AcquirePasswordChangeLock(ctx, userID)
	if err != nil {
		return err
	}
	if !locked {
		return ErrOperationLocked
	}

	// Update the password
	query := `
		UPDATE user_auth
		SET hashed_password = $1
		WHERE id = $2
		RETURNING id
	`

	var id string
	err = r.db.QueryRowContext(ctx, query, hashedPassword, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	// Release the lock after operation is complete
	_ = r.ReleasePasswordChangeLock(ctx, userID)

	return nil
}

func (r *authRepository) StoreRefreshToken(ctx context.Context, userID, refreshToken string) error {
	key := refreshTokenKey(userID, refreshToken)
	return r.redis.Set(ctx, key, "1", refreshTokenTTL)
}

func (r *authRepository) ValidateRefreshToken(ctx context.Context, userID, refreshToken string) (bool, error) {
	key := refreshTokenKey(userID, refreshToken)
	val, err := r.redis.Get(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return false, ErrTokenNotFound
		}
		return false, err
	}
	return val == "1", nil
}

func (r *authRepository) DeleteRefreshToken(ctx context.Context, userID, refreshToken string) error {
	key := refreshTokenKey(userID, refreshToken)
	return r.redis.Delete(ctx, key)
}

func (r *authRepository) AcquirePasswordChangeLock(ctx context.Context, userID string) (bool, error) {
	key := passwordLockKey(userID)
	
	// Try to set the lock with NX (only if does not exist)
	success, err := r.redis.SetNX(ctx, key, "1", passwordLockTTL)
	if err != nil {
		return false, err
	}
	
	return success, nil
}

func (r *authRepository) ReleasePasswordChangeLock(ctx context.Context, userID string) error {
	key := passwordLockKey(userID)
	return r.redis.Delete(ctx, key)
}

// Helper functions for Redis keys
func refreshTokenKey(userID, token string) string {
	return fmt.Sprintf("%s%s:%s", refreshTokenPrefix, userID, token)
}

func passwordLockKey(userID string) string {
	return fmt.Sprintf("%s%s", passwordLockPrefix, userID)
}