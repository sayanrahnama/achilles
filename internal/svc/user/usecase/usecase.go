package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hailsayan/achilles/internal/svc/user/constant"
	"github.com/hailsayan/achilles/internal/svc/user/dto"
	"github.com/hailsayan/achilles/internal/svc/user/entity"
	"github.com/hailsayan/achilles/internal/svc/user/grpcerror"
	"github.com/hailsayan/achilles/internal/svc/user/repository"
)

type UserUseCase interface {
	CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.CreateUserResponse, error)
	GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.GetUserResponse, error)
	UpdateUser(ctx context.Context, req *dto.UpdateUserRequest) (*dto.UpdateUserResponse, error)
	DeleteUser(ctx context.Context, req *dto.DeleteUserRequest) (*dto.DeleteUserResponse, error)
}

type userUseCaseImpl struct {
	dataStore repository.DataStore
	redisRepo repository.RedisRepository
}

func NewUserUseCase(
	dataStore repository.DataStore,
	redisRepo repository.RedisRepository,
) UserUseCase {
	return &userUseCaseImpl{
		dataStore: dataStore,
		redisRepo: redisRepo,
	}
}

func (u *userUseCaseImpl) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.CreateUserResponse, error) {
	res := new(dto.CreateUserResponse)
	err := u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepository := ds.UserRepository()

		normalizedEmail := strings.ToLower(strings.TrimSpace(req.Email))

		existingUser, err := userRepository.GetByEmail(ctx, normalizedEmail)
		if err != nil {
			return err
		}
		if existingUser != nil {
			return grpcerror.NewEmailExistsError()
		}

		userID := uuid.New().String()
		now := time.Now().UTC()

		user := &entity.User{
			ID:        userID,
			Email:     normalizedEmail,
			FirstName: strings.TrimSpace(req.FirstName),
			LastName:  strings.TrimSpace(req.LastName),
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := userRepository.CreateUser(ctx, user); err != nil {
			return err
		}

		res = dto.ToCreateUserResponse(user)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (u *userUseCaseImpl) GetUser(ctx context.Context, req *dto.GetUserRequest) (*dto.GetUserResponse, error) {
	cacheKey := fmt.Sprintf(constant.UserCachePrefix, req.ID)

	if cachedData, err := u.redisRepo.Get(ctx, cacheKey); err == nil && cachedData != "" {
		var user entity.User
		if err := json.Unmarshal([]byte(cachedData), &user); err == nil {
			return dto.ToGetUserResponse(&user), nil
		}
	}

	res := new(dto.GetUserResponse)
	err := u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepository := ds.UserRepository()

		user, err := userRepository.GetByUserID(ctx, req.ID)
		if err != nil {
			return err
		}

		if user == nil {
			return grpcerror.NewUserNotFoundError()
		}

		if userData, err := json.Marshal(user); err == nil {
			u.redisRepo.Set(ctx, cacheKey, string(userData), constant.UserCacheTTL)
		}

		res = dto.ToGetUserResponse(user)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (u *userUseCaseImpl) UpdateUser(ctx context.Context, req *dto.UpdateUserRequest) (*dto.UpdateUserResponse, error) {
	res := new(dto.UpdateUserResponse)
	err := u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepository := ds.UserRepository()

		existingUser, err := userRepository.GetByUserID(ctx, req.ID)
		if err != nil {
			return err
		}
		if existingUser == nil {
			return grpcerror.NewUserNotFoundError()
		}

		updatedUser := &entity.User{
			ID:        existingUser.ID,
			Email:     existingUser.Email,
			FirstName: existingUser.FirstName,
			LastName:  existingUser.LastName,
			CreatedAt: existingUser.CreatedAt,
			UpdatedAt: time.Now().UTC(),
		}

		if req.Email != nil {
			normalizedEmail := strings.ToLower(strings.TrimSpace(*req.Email))
			existingEmailUser, err := userRepository.GetByEmail(ctx, normalizedEmail)
			if err != nil {
				return err
			}
			if existingEmailUser != nil && existingEmailUser.ID != req.ID {
				return grpcerror.NewEmailExistsError()
			}
			updatedUser.Email = normalizedEmail
		}

		if req.FirstName != nil {
			updatedUser.FirstName = strings.TrimSpace(*req.FirstName)
		}

		if req.LastName != nil {
			updatedUser.LastName = strings.TrimSpace(*req.LastName)
		}

		if err := userRepository.UpdateUser(ctx, updatedUser); err != nil {
			return err
		}

		cacheKey := fmt.Sprintf(constant.UserCachePrefix, req.ID)
		u.redisRepo.Delete(ctx, cacheKey)

		res = dto.ToUpdateUserResponse(updatedUser)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (u *userUseCaseImpl) DeleteUser(ctx context.Context, req *dto.DeleteUserRequest) (*dto.DeleteUserResponse, error) {
	res := new(dto.DeleteUserResponse)
	err := u.dataStore.Atomic(ctx, func(ds repository.DataStore) error {
		userRepository := ds.UserRepository()

		existingUser, err := userRepository.GetByUserID(ctx, req.ID)
		if err != nil {
			return err
		}
		if existingUser == nil {
			return grpcerror.NewUserNotFoundError()
		}

		if err := userRepository.DeleteUserByID(ctx, req.ID); err != nil {
			return err
		}

		cacheKey := fmt.Sprintf(constant.UserCachePrefix, req.ID)
		u.redisRepo.Delete(ctx, cacheKey)

		res = dto.ToDeleteUserResponse(true, constant.UserDeletedSuccessfully)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}
