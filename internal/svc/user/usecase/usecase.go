package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hailsayan/achilles/internal/svc/user/model"
	"github.com/hailsayan/achilles/internal/svc/user/repository"
)

const (
	UserCachePrefix = "user:%s"   // user:id
	UserCacheTTL    = time.Hour * 24
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserUseCase interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id string) error
}


type userUseCase struct {
	dataStore DataStore
	redisRepo RedisRepository
}

func NewUserUseCase(dataStore DataStore, redisRepo RedisRepository) UserUseCase {
	return &userUseCase{
		dataStore: dataStore,
		redisRepo: redisRepo,
	}
}

func (u *userUseCase) CreateUser(ctx context.Context, user *model.User) error {
	return u.dataStore.Atomic(ctx, func(ds DataStore) error {
		userRepo := ds.UserRepository()
		return userRepo.Create(ctx, user)
	})
}

func (u *userUseCase) GetUser(ctx context.Context, id string) (*model.User, error) {
	// Try to get user from cache first
	cacheKey := fmt.Sprintf(UserCachePrefix, id)
	
	// Here you would typically deserialize cached user data
	// For simplicity, we're just checking if it exists in cache
	_, err := u.redisRepo.Get(ctx, cacheKey)
	if err == nil {
		// In a real implementation, we would deserialize and return the cached user
		// But for simplicity, we'll just proceed to get it from the database
	}
	
	var user *model.User
	err = u.dataStore.Atomic(ctx, func(ds DataStore) error {
		userRepo := ds.UserRepository()
		var err error
		user, err = userRepo.GetByID(ctx, id)
		return err
	})
	
	if err != nil {
		return nil, err
	}
	
	// Cache the user for future requests
	if user != nil {
		// In a real implementation, we would serialize the user object
		// For simplicity, we're just storing the ID
		err = u.redisRepo.Set(ctx, cacheKey, user.ID, UserCacheTTL)
		if err != nil {
			// Log the error but continue since caching failure is not critical
			fmt.Printf("Failed to cache user: %v\n", err)
		}
	}
	
	return user, nil
}

func (u *userUseCase) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user *model.User
	err := u.dataStore.Atomic(ctx, func(ds DataStore) error {
		userRepo := ds.UserRepository()
		var err error
		user, err = userRepo.GetByUsername(ctx, username)
		return err
	})
	
	if err != nil {
		return nil, err
	}
	
	// Cache the user's ID for future requests
	if user != nil {
		cacheKey := fmt.Sprintf(UserCachePrefix, user.ID)
		err = u.redisRepo.Set(ctx, cacheKey, user.ID, UserCacheTTL)
		if err != nil {
			// Log the error but continue since caching failure is not critical
			fmt.Printf("Failed to cache user: %v\n", err)
		}
	}
	
	return user, nil
}

func (u *userUseCase) UpdateUser(ctx context.Context, user *model.User) error {
	err := u.dataStore.Atomic(ctx, func(ds DataStore) error {
		userRepo := ds.UserRepository()
		return userRepo.Update(ctx, user)
	})
	
	if err != nil {
		return err
	}
	
	// Invalidate cache after update
	cacheKey := fmt.Sprintf(UserCachePrefix, user.ID)
	if err := u.redisRepo.Delete(ctx, cacheKey); err != nil {
		// Log the error but continue since cache invalidation failure is not critical
		fmt.Printf("Failed to invalidate user cache: %v\n", err)
	}
	
	return nil
}

func (u *userUseCase) DeleteUser(ctx context.Context, id string) error {
	err := u.dataStore.Atomic(ctx, func(ds DataStore) error {
		userRepo := ds.UserRepository()
		return userRepo.Delete(ctx, id)
	})
	
	if err != nil {
		return err
	}
	
	// Invalidate cache after deletion
	cacheKey := fmt.Sprintf(UserCachePrefix, id)
	if err := u.redisRepo.Delete(ctx, cacheKey); err != nil {
		// Log the error but continue since cache invalidation failure is not critical
		fmt.Printf("Failed to invalidate user cache: %v\n", err)
	}
	
	return nil
}