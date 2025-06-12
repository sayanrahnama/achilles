package dto

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	UserID       string `json:"user_id"`
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

type RegisterResponse struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type ValidateTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

type ValidateTokenResponse struct {
	IsValid bool   `json:"is_valid"`
	UserID  string `json:"user_id"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type LogoutRequest struct {
	UserID       string `json:"user_id" validate:"required"`
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ChangePasswordRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

type ChangePasswordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}