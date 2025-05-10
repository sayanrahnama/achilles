package model

import "time"

// UserAuth represents the user authentication data
type UserAuth struct {
	ID             string    `json:"id"`
	HashedPassword string    `json:"hashed_password"`
}

// Token represents authentication tokens
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id"`
}

// AuthRequest represents login/register request data
type AuthRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// PasswordChange represents password change request data
type PasswordChange struct {
	UserID      string `json:"user_id"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}