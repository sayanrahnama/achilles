package dto

import (
	"time"

	"github.com/hailsayan/achilles/internal/svc/user/entity"
)

type CreateUserRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Email     string `json:"email" validate:"required,email"`
	FirstName string `json:"first_name" validate:"required,max=64"`
	LastName  string `json:"last_name" validate:"required,max=64"`
}

type GetUserRequest struct {
	ID string `json:"id" validate:"required"`
}

type GetUserByUsernameRequest struct {
	Username string `json:"username" validate:"required"`
}

type UpdateUserRequest struct {
	ID        string  `json:"id" validate:"required"`
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,max=64"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,max=64"`
}

type DeleteUserRequest struct {
	ID string `json:"id" validate:"required"`
}

type CreateUserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetUserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateUserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeleteUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func ToCreateUserResponse(user *entity.User) *CreateUserResponse {
	return &CreateUserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func ToGetUserResponse(user *entity.User) *GetUserResponse {
	return &GetUserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func ToUpdateUserResponse(user *entity.User) *UpdateUserResponse {
	return &UpdateUserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func ToDeleteUserResponse(success bool, message string) *DeleteUserResponse {
	return &DeleteUserResponse{
		Success: success,
		Message: message,
	}
}