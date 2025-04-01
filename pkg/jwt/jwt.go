package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hailsayan/achilles/pkg/logger"
	"go.uber.org/zap"
)

type Claims struct {
	UserID   string            `json:"user_id"`
	Username string            `json:"username"`
	Custom   map[string]string `json:"custom,omitempty"`
}

type TokenService interface {
	GenerateAccessToken(claims Claims) (string, time.Time, error)
	GenerateRefreshToken(userID string) (string, time.Time, error)
	ValidateToken(tokenString string) (*Claims, error)
	GetTokenExpiration(tokenString string) (time.Time, error)
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
	Issuer        string
}

type JWTService struct {
	config JWTConfig
	logger logger.Logger
}

func NewJWTService(config JWTConfig, logger logger.Logger) TokenService {
	return &JWTService{
		config: config,
		logger: logger,
	}
}

func (s *JWTService) GenerateAccessToken(claims Claims) (string, time.Time, error) {
	now := time.Now()
	expirationTime := now.Add(s.config.AccessExpiry)

	s.logger.Debug("Generating access token",
		zap.String("user_id", claims.UserID),
		zap.String("username", claims.Username),
		zap.Time("expiration", expirationTime),
	)

	// Create the JWT token claims
	tokenClaims := jwt.MapClaims{
		"user_id":  claims.UserID,
		"username": claims.Username,
		"iss":      s.config.Issuer,
		"iat":      now.Unix(),
		"exp":      expirationTime.Unix(),
		"type":     "access",
	}

	// Add custom claims if they exist
	if claims.Custom != nil {
		for key, value := range claims.Custom {
			tokenClaims[key] = value
		}
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)

	// Generate the encoded token
	tokenString, err := token.SignedString([]byte(s.config.AccessSecret))
	if err != nil {
		s.logger.Error("Failed to generate access token",
			zap.String("user_id", claims.UserID),
			zap.Error(err),
		)
		return "", time.Time{}, err
	}

	s.logger.Info("Access token generated successfully",
		zap.String("user_id", claims.UserID),
		zap.Time("expiration", expirationTime),
	)

	return tokenString, expirationTime, nil
}

func (s *JWTService) GenerateRefreshToken(userID string) (string, time.Time, error) {
	now := time.Now()
	expirationTime := now.Add(s.config.RefreshExpiry)

	s.logger.Debug("Generating refresh token",
		zap.String("user_id", userID),
		zap.Time("expiration", expirationTime),
	)

	// Create refresh token claims
	tokenClaims := jwt.MapClaims{
		"user_id": userID,
		"iss":     s.config.Issuer,
		"iat":     now.Unix(),
		"exp":     expirationTime.Unix(),
		"type":    "refresh",
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims)

	// Generate the encoded token
	tokenString, err := token.SignedString([]byte(s.config.RefreshSecret))
	if err != nil {
		s.logger.Error("Failed to generate refresh token",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return "", time.Time{}, err
	}

	s.logger.Info("Refresh token generated successfully",
		zap.String("user_id", userID),
		zap.Time("expiration", expirationTime),
	)

	return tokenString, expirationTime, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	s.logger.Debug("Validating token")

	// First try to validate as access token
	claims, err := s.validateWithSecret(tokenString, s.config.AccessSecret)
	if err == nil {
		s.logger.Info("Access token validated successfully",
			zap.String("user_id", claims.UserID),
		)
		return claims, nil
	}

	// Then try as refresh token
	claims, err = s.validateWithSecret(tokenString, s.config.RefreshSecret)
	if err == nil {
		s.logger.Info("Refresh token validated successfully",
			zap.String("user_id", claims.UserID),
		)
		return claims, nil
	}

	s.logger.Warn("Token validation failed", zap.Error(err))
	return nil, fmt.Errorf("invalid token: %w", err)
}

func (s *JWTService) validateWithSecret(tokenString, secret string) (*Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Extract claims
	userID, _ := claims["user_id"].(string)
	username, _ := claims["username"].(string)

	// Extract custom claims
	customClaims := make(map[string]string)
	for key, value := range claims {
		// Skip standard claims
		if key == "user_id" || key == "username" || key == "iss" ||
			key == "iat" || key == "exp" || key == "type" {
			continue
		}

		if strValue, ok := value.(string); ok {
			customClaims[key] = strValue
		}
	}

	return &Claims{
		UserID:   userID,
		Username: username,
		Custom:   customClaims,
	}, nil
}

func (s *JWTService) GetTokenExpiration(tokenString string) (time.Time, error) {
	s.logger.Debug("Getting token expiration")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Try access token first
		return []byte(s.config.AccessSecret), nil
	})

	if err != nil {
		// Try with refresh token secret
		token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.config.RefreshSecret), nil
		})

		if err != nil {
			s.logger.Error("Failed to parse token for expiration", zap.Error(err))
			return time.Time{}, err
		}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		s.logger.Error("Invalid token claims when getting expiration")
		return time.Time{}, errors.New("invalid token claims")
	}

	// Get expiration timestamp
	exp, ok := claims["exp"].(float64)
	if !ok {
		s.logger.Error("Invalid expiration claim")
		return time.Time{}, errors.New("invalid expiration claim")
	}

	expTime := time.Unix(int64(exp), 0)

	s.logger.Debug("Token expiration retrieved successfully",
		zap.Time("expiration", expTime))

	return expTime, nil
}
