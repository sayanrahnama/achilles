package handler

import (
	"context"
	"errors"
	"time"

	"github.com/hailsayan/achilles/internal/pkg/logger"
	userpb "github.com/hailsayan/achilles/internal/svc/user/pb/user"
	"github.com/hailsayan/achilles/internal/svc/user/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserHandler struct {
	userpb  userpb.UnimplementedUserServiceServer
	usecase usecase.UserUseCase
	logger  logger.Logger
}

func NewUserHandler(usecase usecase.UserUseCase, logger logger.Logger) *UserHandler {
	return &UserHandler{
		usecase: usecase,
		logger:  logger,
	}
}

func (h *UserHandler) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.UserResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	user, err := h.usecase.CreateUser(ctx, req.Id, req.Username, req.Email, req.FirstName, req.LastName)
	if err != nil {
		if errors.Is(err, usecase.ErrMissingRequiredData) {
			return nil, status.Errorf(codes.InvalidArgument, "missing required fields")
		}
		// usecase will already translate ErrUserExists to the appropriate gRPC error
		return nil, err
	}

	return userToProto(user), nil
}

func (h *UserHandler) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.UserResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
	}

	user, err := h.usecase.GetUser(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		h.logger.Error("Failed to get user", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}

	return userToProto(user), nil
}

func (h *UserHandler) GetUserByUsername(ctx context.Context, req *userpb.GetUserByUsernameRequest) (*userpb.UserResponse, error) {
	if req == nil || req.Username == "" {
		return nil, status.Errorf(codes.InvalidArgument, "username is required")
	}

	user, err := h.usecase.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		if errors.Is(err, usecase.ErrInvalidUsername) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid username")
		}
		h.logger.Error("Failed to get user by username", "error", err, "username", req.Username)
		return nil, status.Errorf(codes.Internal, "failed to get user")
	}

	return userToProto(user), nil
}

func (h *UserHandler) GetUsers(ctx context.Context, req *userpb.GetUsersRequest) (*userpb.GetUsersResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}

	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10 // Default page size
	}

	users, total, err := h.usecase.GetUsers(ctx, page, pageSize, req.SortBy, req.SortDesc)
	if err != nil {
		h.logger.Error("Failed to get users", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get users")
	}

	userResponses := make([]*userpb.UserResponse, 0, len(users))
	for _, user := range users {
		userResponses = append(userResponses, userToProto(user))
	}

	return &userpb.GetUsersResponse{
		Users:    userResponses,
		Total:    int32(total),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UserResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
	}

	user, err := h.usecase.UpdateUser(ctx, req.UserId, req.Email, req.FirstName, req.LastName)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		h.logger.Error("Failed to update user", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to update user")
	}

	return userToProto(user), nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	if req == nil || req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
	}

	err := h.usecase.DeleteUser(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, usecase.ErrUserNotFound) {
			return &userpb.DeleteUserResponse{
				Success: false,
				Message: "user not found",
			}, nil
		}
		h.logger.Error("Failed to delete user", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to delete user")
	}

	return &userpb.DeleteUserResponse{
		Success: true,
		Message: "user deleted successfully",
	}, nil
}

// Helper function to convert user model to proto response
func userToProto(user interface{}) *userpb.UserResponse {
	if user == nil {
		return nil
	}

	// Try to convert to usecase.UserResponse (if using the helper function)
	if u, ok := user.(*usecase.UserResponse); ok && u != nil {
		return &userpb.UserResponse{
			Id:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			CreatedAt: u.CreatedAt.Seconds,
			UpdatedAt: u.UpdatedAt.Seconds,
		}
	}

	// Try direct model conversion
	if u, ok := user.(struct {
		ID        string
		Username  string
		Email     string
		FirstName string
		LastName  string
		CreatedAt time.Time
		UpdatedAt time.Time
	}); ok {
		return &userpb.UserResponse{
			Id:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			CreatedAt: timestamppb.New(u.CreatedAt).Seconds,
			UpdatedAt: timestamppb.New(u.UpdatedAt).Seconds,
		}
	}

	return nil
}
