package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hailsayan/achilles/auth/model"
	"github.com/hailsayan/achilles/auth/repository"
	"github.com/hailsayan/achilles/pkg/logger"
	"github.com/hailsayan/achilles/pkg/utils/encryptutils"
	"github.com/hailsayan/achilles/pkg/utils/jwtutils"
	usermodel "github.com/hailsayan/achilles/user/model"
	userrepo "github.com/hailsayan/achilles/user/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid token")
	ErrPasswordMismatch   = errors.New("old password does not match")
)

type AuthService interface {
	Register(ctx context.Context, req *model.AuthRequest) (string, error)
	Login(ctx context.Context, username, password string) (*model.Token, error)
	ValidateToken(ctx context.Context, token string) (bool, string, error)
	RefreshToken(ctx context.Context, refreshToken string) (*model.Token, error)
	Logout(ctx context.Context, userID, refreshToken string) error
	ChangePassword(ctx context.Context, req *model.PasswordChange) error
}

type authService struct {
	authRepo repository.AuthRepository
	userRepo userrepo.UserRepository
	jwtUtil  jwtutils.JwtUtil
	hasher   encryptutils.Hasher
	logger   logger.Logger
}

func NewAuthService(
	authRepo repository.AuthRepository,
	userRepo userrepo.UserRepository,
	jwtUtil jwtutils.JwtUtil,
	hasher encryptutils.Hasher,
	logger logger.Logger,
) AuthService {
	return &authService{
		authRepo: authRepo,
		userRepo: userRepo,
		jwtUtil:  jwtUtil,
		hasher:   hasher,
		logger:   logger,
	}
}

func (s *authService) Register(ctx context.Context, req *model.AuthRequest) (string, error) {
	// Generate UUID for the new user
	userID := uuid.New().String()

	// Create user object
	user := &usermodel.User{
		ID:        userID,
		Username:  req.Username,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	// Try to create user first
	err := s.userRepo.Create(ctx, user)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserExists) {
			return "", ErrUserExists
		}
		return "", err
	}

	// Hash the password
	hashedPassword, err := s.hasher.Hash(req.Password)
	if err != nil {
		// If password hashing fails, try to rollback user creation
		// This is a best effort, and may not always succeed
		_ = s.userRepo.Delete(ctx, userID)
		return "", err
	}

	// Store user auth data
	userAuth := &model.UserAuth{
		ID:             userID,
		HashedPassword: hashedPassword,
	}

	err = s.authRepo.StoreUserAuth(ctx, userAuth)
	if err != nil {
		// If auth creation fails, try to rollback user creation
		_ = s.userRepo.Delete(ctx, userID)
		return "", err
	}

	return userID, nil
}

func (s *authService) Login(ctx context.Context, username, password string) (*model.Token, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, userrepo.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Get user auth info
	userAuth, err := s.authRepo.GetUserAuth(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	// Verify password
	if !s.hasher.Check(password, userAuth.HashedPassword) {
		return nil, ErrInvalidCredentials
	}

	// Generate access token
	accessToken, expiresAt, err := s.jwtUtil.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := s.jwtUtil.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	// Store refresh token in Redis
	err = s.authRepo.StoreRefreshToken(ctx, user.ID, refreshToken)
	if err != nil {
		return nil, err
	}

	// Create and return token model
	token := &model.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		UserID:       user.ID,
	}

	return token, nil
}

func (s *authService) ValidateToken(ctx context.Context, token string) (bool, string, error) {
	// Parse and validate the token
	claims, err := s.jwtUtil.ValidateToken(token)
	if err != nil {
		return false, "", ErrInvalidToken
	}

	// Ensure it's an access token
	if claims.TokenType != "access" {
		return false, "", ErrInvalidToken
	}

	// Return validation result
	return true, claims.UserID, nil
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*model.Token, error) {
	// Parse refresh token to get claims
	claims, err := s.jwtUtil.ValidateToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Ensure it's a refresh token
	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	userID := claims.UserID

	// Validate refresh token in Redis
	valid, err := s.authRepo.ValidateRefreshToken(ctx, userID, refreshToken)
	if err != nil || !valid {
		return nil, ErrInvalidToken
	}

	// Get user for username
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Delete the old refresh token
	err = s.authRepo.DeleteRefreshToken(ctx, userID, refreshToken)
	if err != nil {
		s.logger.Warn("Failed to delete old refresh token", "error", err)
		// Continue anyway, this is not critical
	}

	// Generate new access token
	accessToken, expiresAt, err := s.jwtUtil.GenerateAccessToken(userID, user.Username)
	if err != nil {
		return nil, err
	}

	// Generate new refresh token
	newRefreshToken, err := s.jwtUtil.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	// Store new refresh token
	err = s.authRepo.StoreRefreshToken(ctx, userID, newRefreshToken)
	if err != nil {
		return nil, err
	}

	// Create token response
	token := &model.Token{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		UserID:       userID,
	}

	return token, nil
}

func (s *authService) Logout(ctx context.Context, userID, refreshToken string) error {
	// Invalidate refresh token
	return s.authRepo.DeleteRefreshToken(ctx, userID, refreshToken)
}

func (s *authService) ChangePassword(ctx context.Context, req *model.PasswordChange) error {
	// Get user auth info
	userAuth, err := s.authRepo.GetUserAuth(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	// Verify old password
	if !s.hasher.Check(req.OldPassword, userAuth.HashedPassword) {
		return ErrPasswordMismatch
	}

	// Hash new password
	hashedPassword, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return err
	}

	// Update password in database
	return s.authRepo.UpdateUserPassword(ctx, req.UserID, hashedPassword)
}
