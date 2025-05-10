package handler

import (
	"context"
	"errors"

	"github.com/hailsayan/achilles/auth/model"
	"github.com/hailsayan/achilles/auth/service"
	"github.com/hailsayan/achilles/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	authpb.UnimplementedAuthServiceServer
	authService service.AuthService
	logger      logger.Logger
}

func NewAuthHandler(authService service.AuthService, logger logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

func (h *AuthHandler) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	authReq := &model.AuthRequest{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	userID, err := h.authService.Register(ctx, authReq)
	if err != nil {
		h.logger.Error("Failed to register user", "error", err)

		if errors.Is(err, service.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "User already exists")
		}

		return nil, status.Error(codes.Internal, "Failed to register user")
	}

	return &authpb.RegisterResponse{
		UserId:  userID,
		Message: "User registered successfully",
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	token, err := h.authService.Login(ctx, req.Username, req.Password)
	if err != nil {
		h.logger.Error("Login failed", "error", err)

		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "Invalid credentials")
		}

		return nil, status.Error(codes.Internal, "Failed to login")
	}

	return &authpb.LoginResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt.Unix(),
		UserId:       token.UserID,
	}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	isValid, userID, err := h.authService.ValidateToken(ctx, req.Token)
	if err != nil {
		h.logger.Error("Token validation failed", "error", err)

		if errors.Is(err, service.ErrInvalidToken) {
			return &authpb.ValidateTokenResponse{
				IsValid: false,
			}, nil
		}

		return nil, status.Error(codes.Internal, "Failed to validate token")
	}

	return &authpb.ValidateTokenResponse{
		IsValid: isValid,
		UserId:  userID,
	}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	token, err := h.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		h.logger.Error("Token refresh failed", "error", err)

		if errors.Is(err, service.ErrInvalidToken) {
			return nil, status.Error(codes.Unauthenticated, "Invalid refresh token")
		}

		return nil, status.Error(codes.Internal, "Failed to refresh token")
	}

	return &authpb.RefreshTokenResponse{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.ExpiresAt.Unix(),
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	err := h.authService.Logout(ctx, req.UserId, req.RefreshToken)
	if err != nil {
		h.logger.Error("Logout failed", "error", err)
		return nil, status.Error(codes.Internal, "Failed to logout")
	}

	return &authpb.LogoutResponse{
		Success: true,
	}, nil
}

func (h *AuthHandler) ChangePassword(ctx context.Context, req *authpb.ChangePasswordRequest) (*authpb.ChangePasswordResponse, error) {
	passwordChange := &model.PasswordChange{
		UserID:      req.UserId,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	err := h.authService.ChangePassword(ctx, passwordChange)
	if err != nil {
		h.logger.Error("Password change failed", "error", err)

		switch {
		case errors.Is(err, service.ErrUserNotFound):
			return nil, status.Error(codes.NotFound, "User not found")
		case errors.Is(err, service.ErrPasswordMismatch):
			return nil, status.Error(codes.PermissionDenied, "Old password is incorrect")
		default:
			return nil, status.Error(codes.Internal, "Failed to change password")
		}
	}

	return &authpb.ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully",
	}, nil
}
