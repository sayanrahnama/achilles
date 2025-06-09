package handler

import (
	"context"
	
	"github.com/hailsayan/achilles/internal/svc/user/dto"
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
	createReq := &dto.CreateUserRequest{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	res, err := h.userUseCase.CreateUser(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return h.toUserResponse(res.ID, res.Email, res.FirstName, res.LastName, res.CreatedAt.Unix(), res.UpdatedAt.Unix()), nil
}

func (h *UserHandler) GetUserByID(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	getUserReq := &dto.GetUserRequest{
		ID: req.UserId,
	}

	res, err := h.userUseCase.GetUser(ctx, getUserReq)
	if err != nil {
		return nil, err
	}

	return h.toUserResponse(res.ID, res.Email, res.FirstName, res.LastName, res.CreatedAt.Unix(), res.UpdatedAt.Unix()), nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UserResponse, error) {
	updateReq := &dto.UpdateUserRequest{
		ID: req.UserId,
	}

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

	return h.toUserResponse(res.ID, res.Email, res.FirstName, res.LastName, res.CreatedAt.Unix(), res.UpdatedAt.Unix()), nil
}

func (h *UserHandler) DeleteUserByID(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	deleteReq := &dto.DeleteUserRequest{
		ID: req.UserId,
	}

	res, err := h.userUseCase.DeleteUser(ctx, deleteReq)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteUserResponse{
		Success: res.Success,
		Message: res.Message,
	}, nil
}

func (h *UserHandler) toUserResponse(id, email, firstName, lastName string, createdAt, updatedAt int64) *pb.UserResponse {
	return &pb.UserResponse{
		Id:        id,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}