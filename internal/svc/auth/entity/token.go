package entity

import "time"

type Token struct {
	UserID       string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}
