package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hailsayan/achilles/internal/pkg/logger"
	"github.com/hailsayan/achilles/internal/svc/user/model"
	"github.com/hailsayan/achilles/internal/svc/user/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrInvalidUserID       = errors.New("invalid user ID")
	ErrInvalidUsername     = errors.New("invalid username")
	ErrUserNotFound        = errors.New("user not found")
	ErrMissingRequiredData = errors.New("missing required data")
)

type UserUseCase interface {
	CreateUser(ctx context.Context, id, username, email, firstName, lastName string) (*model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUsers(ctx context.Context, page, pageSize int, sortBy string, sortDesc bool) ([]*model.User, int, error)
	UpdateUser(ctx context.Context, id, email, firstName, lastName string) (*model.User, error)
	DeleteUser(ctx context.Context, id string) error
}

type userUseCase struct {
	repo repository.UserRepository
	l    logger.Logger
}

func NewUserUseCase(repo repository.UserRepository, logger logger.Logger) UserUseCase {
	return &userUseCase{
		repo: repo,
		l:    logger,
	}
}

func (u *userUseCase) CreateUser(ctx context.Context, id, username, email, firstName, lastName string) (*model.User, error) {
	// Validate input
	if err := u.validateCreateUserInput(id, username, email, firstName, lastName); err != nil {
		return nil, err
	}

	// Create user ID if not provided
	userID := id
	if userID == "" {
		userID = uuid.New().String()
	}

	user := &model.User{
		ID:        userID,
		Username:  strings.ToLower(username),
		Email:     strings.ToLower(email),
		FirstName: firstName,
		LastName:  lastName,
	}

	if err := u.repo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			return nil, status.Errorf(codes.AlreadyExists, "user already exists with this username or email")
		}
		u.l.Error("Failed to create user", "error", err)
		return nil, err
	}

	return user, nil
}

func (u *userUseCase) GetUser(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, ErrInvalidUserID
	}

	user, err := u.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		u.l.Error("Failed to get user", "error", err, "id", id)
		return nil, err
	}

	return user, nil
}

func (u *userUseCase) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, ErrInvalidUsername
	}

	// Convert to lowercase to ensure case-insensitive lookup
	username = strings.ToLower(username)

	user, err := u.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		u.l.Error("Failed to get user by username", "error", err, "username", username)
		return nil, err
	}

	return user, nil
}

func (u *userUseCase) GetUsers(ctx context.Context, page, pageSize int, sortBy string, sortDesc bool) ([]*model.User, int, error) {
	users, total, err := u.repo.GetUsers(ctx, page, pageSize, sortBy, sortDesc)
	if err != nil {
		u.l.Error("Failed to get users", "error", err, "page", page, "pageSize", pageSize)
		return nil, 0, err
	}

	return users, total, nil
}

func (u *userUseCase) UpdateUser(ctx context.Context, id, email, firstName, lastName string) (*model.User, error) {
	if id == "" {
		return nil, ErrInvalidUserID
	}

	// Get existing user
	user, err := u.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		u.l.Error("Failed to get user for update", "error", err, "id", id)
		return nil, err
	}

	// Update user fields if provided
	if email != "" {
		user.Email = strings.ToLower(email)
	}

	if firstName != "" {
		user.FirstName = firstName
	}

	if lastName != "" {
		user.LastName = lastName
	}

	// Update the user in the repository
	err = u.repo.Update(ctx, user)
	if err != nil {
		u.l.Error("Failed to update user", "error", err, "id", id)
		return nil, err
	}

	return user, nil
}

func (u *userUseCase) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidUserID
	}

	err := u.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		u.l.Error("Failed to delete user", "error", err, "id", id)
		return err
	}

	return nil
}

func (u *userUseCase) validateCreateUserInput(id, username, email, firstName, lastName string) error {
	if username == "" || email == "" || firstName == "" {
		return ErrMissingRequiredData
	}

	// Additional validations can be added here
	// e.g., check email format, username constraints, etc.

	return nil
}

// Helper function to convert User model to UserResponse
func UserToUserResponse(user *model.User) *UserResponse {
	if user == nil {
		return nil
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}

// UserResponse struct matching the proto definition
type UserResponse struct {
	ID        string
	Username  string
	Email     string
	FirstName string
	LastName  string
	CreatedAt *timestamppb.Timestamp
	UpdatedAt *timestamppb.Timestamp
}
