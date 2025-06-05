package handler

import (
	"context"
	
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hailsayan/achilles/internal/svc/user/dto"
	"github.com/hailsayan/achilles/internal/svc/user/entity"
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

	createReq := &dto.CreateUserRequest{
		Username:  req.Username,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	res, err := h.userUseCase.CreateUser(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return h.createResponseToProto(res), nil
}

func (h *UserHandler) GetUserByID(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	getUserReq := &dto.GetUserRequest{
		ID: req.UserId,
	}

	res, err := h.userUseCase.GetUser(ctx, getUserReq)
	if err != nil {
		return nil, err
	}

	return h.getUserResponseToProto(res), nil
}

func (h *UserHandler) GetUserByUsername(ctx context.Context, req *pb.GetUserByUsernameRequest) (*pb.UserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	getUserReq := &dto.GetUserByUsernameRequest{
		Username: req.Username,
	}

	res, err := h.userUseCase.GetUserByUsername(ctx, getUserReq)
	if err != nil {
		return nil, err
	}

	return h.getUserResponseToProto(res), nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	updateReq := &dto.UpdateUserRequest{
		ID: req.UserId,
	}

	// Only set fields that are provided
	if req.Email != nil {
		updateReq.Email = req.Email
	}
	if req.FirstName != nil {
		updateReq.FirstName = req.FirstName
	}
	if req.LastName != nil {
		updateReq.LastName = req.LastName
	}

	res, err := h.userUseCase.UpdateUser(ctx, updateReq)
	if err != nil {
		return nil, err
	}

	return h.updateResponseToProto(res), nil
}

func (h *UserHandler) DeleteUserByID(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	deleteReq := &dto.DeleteUserRequest{
		ID: req.UserId,
	}

	res, err := h.userUseCase.DeleteUser(ctx, deleteReq)
	if err != nil {
		return &pb.DeleteUserResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &pb.DeleteUserResponse{
		Success: res.Success,
		Message: res.Message,
	}, nil
}

func (h *UserHandler) createResponseToProto(res *dto.CreateUserResponse) *pb.UserResponse {
	if res == nil {
		return nil
	}

	return &pb.UserResponse{
		Id:        res.ID,
		Username:  res.Username,
		Email:     res.Email,
		FirstName: res.FirstName,
		LastName:  res.LastName,
		CreatedAt: res.CreatedAt.Unix(),
		UpdatedAt: res.UpdatedAt.Unix(),
	}
}

func (h *UserHandler) getUserResponseToProto(res *dto.GetUserResponse) *pb.UserResponse {
	if res == nil {
		return nil
	}

	return &pb.UserResponse{
		Id:        res.ID,
		Username:  res.Username,
		Email:     res.Email,
		FirstName: res.FirstName,
		LastName:  res.LastName,
		CreatedAt: res.CreatedAt.Unix(),
		UpdatedAt: res.UpdatedAt.Unix(),
	}
}

func (h *UserHandler) updateResponseToProto(res *dto.UpdateUserResponse) *pb.UserResponse {
	if res == nil {
		return nil
	}

	return &pb.UserResponse{
		Id:        res.ID,
		Username:  res.Username,
		Email:     res.Email,
		FirstName: res.FirstName,
		LastName:  res.LastName,
		CreatedAt: res.CreatedAt.Unix(),
		UpdatedAt: res.UpdatedAt.Unix(),
	}
}

func (h *UserHandler) entityToProto(user *entity.User) *pb.UserResponse {
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