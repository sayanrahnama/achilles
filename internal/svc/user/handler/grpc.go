package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hailsayan/achilles/internal/svc/user/model"
	pb "github.com/hailsayan/achilles/internal/svc/user/pb/user"
	"github.com/hailsayan/achilles/internal/svc/user/usecase"
)

type UserHandler struct {
	pb.UnimplementedUserServiceServer
	userUseCase usecase.UserUseCase
}

func NewUserHandler(userUseCase usecase.UserUseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

func (h *UserHandler) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	user, err := h.userUseCase.CreateUser(ctx, req.Username, req.Email, req.FirstName, req.LastName)
	if err != nil {
		return nil, h.handleError(err)
	}

	return h.modelToProto(user), nil
}

func (h *UserHandler) GetUserByID(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	user, err := h.userUseCase.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, h.handleError(err)
	}

	return h.modelToProto(user), nil
}

func (h *UserHandler) GetUserByUsername(ctx context.Context, req *pb.GetUserByUsernameRequest) (*pb.UserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	user, err := h.userUseCase.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, h.handleError(err)
	}

	return h.modelToProto(user), nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	var email, firstName, lastName *string

	if req.Email != nil {
		email = req.Email
	}
	if req.FirstName != nil {
		firstName = req.FirstName
	}
	if req.LastName != nil {
		lastName = req.LastName
	}

	user, err := h.userUseCase.UpdateUser(ctx, req.UserId, email, firstName, lastName)
	if err != nil {
		return nil, h.handleError(err)
	}

	return h.modelToProto(user), nil
}

func (h *UserHandler) DeleteUserByID(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	err := h.userUseCase.DeleteUser(ctx, req.UserId)
	if err != nil {
		return &pb.DeleteUserResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.DeleteUserResponse{
		Success: true,
		Message: "user deleted successfully",
	}, nil
}
func (h *UserHandler) modelToProto(user *model.User) *pb.UserResponse {
	if user == nil {
		return nil
	}

	return &pb.UserResponse{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}
}

func (h *UserHandler) handleError(err error) error {
	switch {
	case errors.Is(err, usecase.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, usecase.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, "user already exists")
	case errors.Is(err, usecase.ErrUsernameExists):
		return status.Error(codes.AlreadyExists, "username already exists")
	case errors.Is(err, usecase.ErrEmailExists):
		return status.Error(codes.AlreadyExists, "email already exists")
	case errors.Is(err, usecase.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, "invalid input")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}