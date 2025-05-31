package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hailsayan/achilles/internal/svc/user/model"
	"github.com/hailsayan/achilles/internal/svc/user/repository"
)

const (
	UserCachePrefix = "user:%s"
	UserCacheTTL    = time.Hour * 24
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUsernameExists     = errors.New("username already exists")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidInput       = errors.New("invalid input")
)

type UserUseCase interface {
	CreateUser(ctx context.Context, username, email, firstName, lastName string) (*model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	UpdateUser(ctx context.Context, userID string, email, firstName, lastName *string) (*model.User, error)
	DeleteUser(ctx context.Context, id string) error
}

type userUseCase struct {
	dataStore repository.DataStore
	redisRepo repository.RedisRepository
}

func NewUserUseCase(dataStore repository.DataStore, redisRepo repository.RedisRepository) UserUseCase {
	return &userUseCase{
		dataStore: dataStore,
		redisRepo: redisRepo,
	}
}

func (u *userUseCase) CreateUser(ctx context.Context, username, email, firstName, lastName string) (*model.User, error) {
	if err := u.validateCreateUserInput(username, email, firstName, lastName); err != nil {
		return nil, err
	}

	normalizedUsername := strings.ToLower(strings.TrimSpace(username))
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	trimmedFirstName := strings.TrimSpace(firstName)
	trimmedLastName := strings.TrimSpace(lastName)

	userID := uuid.New().String()
	now := time.Now().UTC()

	user := &model.User{
		ID:        userID,
		Username:  normalizedUsername,
		Email:     normalizedEmail,
		FirstName: trimmedFirstName,
		LastName:  trimmedLastName,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepo := ds.UserRepository()

		existingUser, err := userRepo.GetByUsername(ctx, normalizedUsername)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return ErrUsernameExists
		}

		return userRepo.CreateUser(ctx, user)
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *userUseCase) GetUser(ctx context.Context, id string) (*model.User, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidInput
	}

	cacheKey := fmt.Sprintf(UserCachePrefix, id)
	_, err := u.redisRepo.Get(ctx, cacheKey)
	if err == nil {
		// In real implementation, you would deserialize cached user here
		// For now, we'll skip to database
	}

	var user *model.User
	err = u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepo := ds.UserRepository()
		var err error
		user, err = userRepo.GetByUserID(ctx, id)
		return err
	})

	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	if err := u.redisRepo.Set(ctx, cacheKey, user.ID, UserCacheTTL); err != nil {
		fmt.Printf("Failed to cache user: %v\n", err)
	}

	return user, nil
}

func (u *userUseCase) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	if strings.TrimSpace(username) == "" {
		return nil, ErrInvalidInput
	}

	normalizedUsername := strings.ToLower(strings.TrimSpace(username))

	var user *model.User
	err := u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepo := ds.UserRepository()
		var err error
		user, err = userRepo.GetByUsername(ctx, normalizedUsername)
		return err
	})

	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	cacheKey := fmt.Sprintf(UserCachePrefix, user.ID)
	if err := u.redisRepo.Set(ctx, cacheKey, user.ID, UserCacheTTL); err != nil {
		fmt.Printf("Failed to cache user: %v\n", err)
	}

	return user, nil
}

func (u *userUseCase) UpdateUser(ctx context.Context, userID string, email, firstName, lastName *string) (*model.User, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidInput
	}

	var updatedUser *model.User
	err := u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepo := ds.UserRepository()

		existingUser, err := userRepo.GetByUserID(ctx, userID)
		if err != nil {
			return err
		}
		if existingUser == nil {
			return ErrUserNotFound
		}

		updatedUser = &model.User{
			ID:        existingUser.ID,
			Username:  existingUser.Username,
			Email:     existingUser.Email,
			FirstName: existingUser.FirstName,
			LastName:  existingUser.LastName,
			CreatedAt: existingUser.CreatedAt,
			UpdatedAt: time.Now().UTC(),
		}

		if email != nil {
			normalizedEmail := strings.ToLower(strings.TrimSpace(*email))
			if err := u.validateEmail(normalizedEmail); err != nil {
				return err
			}
			updatedUser.Email = normalizedEmail
		}

		if firstName != nil {
			trimmedFirstName := strings.TrimSpace(*firstName)
			if err := u.validateName(trimmedFirstName); err != nil {
				return err
			}
			updatedUser.FirstName = trimmedFirstName
		}

		if lastName != nil {
			trimmedLastName := strings.TrimSpace(*lastName)
			if err := u.validateName(trimmedLastName); err != nil {
				return err
			}
			updatedUser.LastName = trimmedLastName
		}

		return userRepo.UpdateUser(ctx, updatedUser)
	})

	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf(UserCachePrefix, userID)
	if err := u.redisRepo.Delete(ctx, cacheKey); err != nil {
		fmt.Printf("Failed to invalidate user cache: %v\n", err)
	}

	return updatedUser, nil
}

func (u *userUseCase) DeleteUser(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidInput
	}

	err := u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepo := ds.UserRepository()

		existingUser, err := userRepo.GetByUserID(ctx, id)
		if err != nil {
			return err
		}
		if existingUser == nil {
			return ErrUserNotFound
		}

		return userRepo.DeleteUserByID(ctx, id)
	})

	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf(UserCachePrefix, id)
	if err := u.redisRepo.Delete(ctx, cacheKey); err != nil {
		fmt.Printf("Failed to invalidate user cache: %v\n", err)
	}

	return nil
}

func (u *userUseCase) validateCreateUserInput(username, email, firstName, lastName string) error {
	if err := u.validateUsername(username); err != nil {
		return err
	}
	if err := u.validateEmail(email); err != nil {
		return err
	}
	if err := u.validateName(firstName); err != nil {
		return err
	}
	if err := u.validateName(lastName); err != nil {
		return err
	}
	return nil
}

func (u *userUseCase) validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return ErrInvalidInput
	}
	if len(username) < 3 || len(username) > 50 {
		return ErrInvalidInput
	}
	return nil
}

func (u *userUseCase) validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ErrInvalidInput
	}
	if !strings.Contains(email, "@") {
		return ErrInvalidInput
	}
	return nil
}

func (u *userUseCase) validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidInput
	}
	if len(name) > 64 {
		return ErrInvalidInput
	}
	return nil
}