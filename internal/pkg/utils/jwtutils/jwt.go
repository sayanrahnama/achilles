package jwtutils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JwtConfig struct {
	AccessTokenDuration  int      `mapstructure:"access_token_duration"`
	RefreshTokenDuration int      `mapstructure:"refresh_token_duration"`
	SecretKey            string   `mapstructure:"secret_key"`
	Issuer               string   `mapstructure:"issuer"`
	AllowedAlgs          []string `mapstructure:"allowed_algs"`
}

type JwtUtil interface {
	GenerateAccessToken(userID, email string) (string, time.Time, error)
	GenerateRefreshToken(userID string) (string, error)
	ValidateToken(token string) (*JWTClaims, error)
	GetTokenExpiration() time.Time
}

type JWTClaims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TokenType string `json:"token_type"`
}

type jwtUtil struct {
	config *JwtConfig
}

func NewJwtUtil(config *JwtConfig) JwtUtil {
	if config.AllowedAlgs == nil || len(config.AllowedAlgs) == 0 {
		config.AllowedAlgs = []string{"HS256"}
	}
	return &jwtUtil{
		config: config,
	}
}

func (j *jwtUtil) GenerateAccessToken(userID, username string) (string, time.Time, error) {
	currentTime := time.Now()
	expirationTime := currentTime.Add(time.Duration(j.config.AccessTokenDuration) * time.Minute)
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		UserID:   userID,
		Username: username,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(currentTime),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    j.config.Issuer,
		},
	})
	
	signedToken, err := token.SignedString([]byte(j.config.SecretKey))
	if err != nil {
		return "", time.Time{}, err
	}
	
	return signedToken, expirationTime, nil
}

func (j *jwtUtil) GenerateRefreshToken(userID string) (string, error) {
	currentTime := time.Now()
	expirationTime := currentTime.Add(time.Duration(j.config.RefreshTokenDuration) * time.Minute)
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTClaims{
		UserID:   userID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(currentTime),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    j.config.Issuer,
		},
	})
	
	signedToken, err := token.SignedString([]byte(j.config.SecretKey))
	if err != nil {
		return "", err
	}
	
	return signedToken, nil
}

func (j *jwtUtil) ValidateToken(tokenString string) (*JWTClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods(j.config.AllowedAlgs),
		jwt.WithIssuer(j.config.Issuer),
		jwt.WithIssuedAt(),
	)
	
	token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.config.SecretKey), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, errors.New("token is not valid")
}

func (j *jwtUtil) GetTokenExpiration() time.Time {
	return time.Now().Add(time.Duration(j.config.AccessTokenDuration) * time.Minute)
}