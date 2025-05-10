// wallet/service/service.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"wallet/model"
	"wallet/repository"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

const (
	walletCachePrefix = "wallet:"
	walletCacheTTL    = 15 * time.Minute
)

// WalletService defines the business logic interface for wallet operations
type WalletService interface {
	CreateWallet(ctx context.Context, userID string, initialBalance float64) (*model.Wallet, error)
	GetWallet(ctx context.Context, id string) (*model.Wallet, error)
	GetWalletByUserID(ctx context.Context, userID string) (*model.Wallet, error)
	UpdateBalance(ctx context.Context, id string, amount float64) (*model.Wallet, error)
	HasSufficientBalance(ctx context.Context, id string, amount float64) (bool, float64, error)
	BlockWallet(ctx context.Context, id, reason string) (*model.Wallet, error)
	UnblockWallet(ctx context.Context, id string) (*model.Wallet, error)
}

type walletService struct {
	logger         *zap.Logger
	walletRepo     repository.WalletRepository
	redisClient    *redis.Client
}

// NewWalletService creates a new instance of WalletService
func NewWalletService(
	logger *zap.Logger,
	walletRepo repository.WalletRepository,
	redisClient *redis.Client,
) WalletService {
	return &walletService{
		logger:      logger,
		walletRepo:  walletRepo,
		redisClient: redisClient,
	}
}

// CreateWallet creates a new wallet for a user
func (s *walletService) CreateWallet(ctx context.Context, userID string, initialBalance float64) (*model.Wallet, error) {
	s.logger.Info("Creating new wallet", zap.String("user_id", userID), zap.Float64("initial_balance", initialBalance))
	
	wallet := &model.Wallet{
		UserID:  userID,
		Balance: initialBalance,
	}
	
	createdWallet, err := s.walletRepo.Create(ctx, wallet)
	if err != nil {
		s.logger.Error("Failed to create wallet", zap.Error(err), zap.String("user_id", userID))
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}
	
	// Cache the wallet
	if err := s.cacheWallet(ctx, createdWallet); err != nil {
		s.logger.Warn("Failed to cache wallet", zap.Error(err), zap.String("wallet_id", createdWallet.ID))
	}
	
	return createdWallet, nil
}

// GetWallet fetches a wallet by its ID, using cache if available
func (s *walletService) GetWallet(ctx context.Context, id string) (*model.Wallet, error) {
	// Try to get from cache first
	wallet, err := s.getWalletFromCache(ctx, id)
	if err == nil {
		return wallet, nil
	}
	
	// Cache miss, get from database
	s.logger.Debug("Cache miss for wallet", zap.String("wallet_id", id))
	wallet, err = s.walletRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			s.logger.Info("Wallet not found", zap.String("wallet_id", id))
			return nil, err
		}
		s.logger.Error("Failed to get wallet", zap.Error(err), zap.String("wallet_id", id))
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	
	// Cache the wallet for future use
	if err := s.cacheWallet(ctx, wallet); err != nil {
		s.logger.Warn("Failed to cache wallet", zap.Error(err), zap.String("wallet_id", id))
	}
	
	return wallet, nil
}

// GetWalletByUserID fetches a wallet by its owner's user ID
func (s *walletService) GetWalletByUserID(ctx context.Context, userID string) (*model.Wallet, error) {
	// Get from database
	wallet, err := s.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			s.logger.Info("Wallet not found for user", zap.String("user_id", userID))
			return nil, err
		}
		s.logger.Error("Failed to get wallet by user ID", zap.Error(err), zap.String("user_id", userID))
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	
	// Cache the wallet for future use
	if err := s.cacheWallet(ctx, wallet); err != nil {
		s.logger.Warn("Failed to cache wallet", zap.Error(err), zap.String("wallet_id", wallet.ID))
	}
	
	return wallet, nil
}

// UpdateBalance updates a wallet's balance
func (s *walletService) UpdateBalance(ctx context.Context, id string, amount float64) (*model.Wallet, error) {
	s.logger.Info("Updating wallet balance", 
		zap.String("wallet_id", id), 
		zap.Float64("amount", amount),
	)
	
	// Perform the update
	updatedWallet, err := s.walletRepo.UpdateBalance(ctx, id, amount)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			s.logger.Info("Wallet not found for update", zap.String("wallet_id", id))
			return nil, err
		}
		if err == repository.ErrInsufficientFunds {
			s.logger.Info("Insufficient funds for wallet update", 
				zap.String("wallet_id", id),
				zap.Float64("amount", amount),
			)
			return nil, err
		}
		s.logger.Error("Failed to update wallet balance", 
			zap.Error(err), 
			zap.String("wallet_id", id),
			zap.Float64("amount", amount),
		)
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}
	
	// Update the cache
	if err := s.cacheWallet(ctx, updatedWallet); err != nil {
		s.logger.Warn("Failed to update wallet cache", zap.Error(err), zap.String("wallet_id", id))
	}
	
	// Clear any related cache entries
	s.invalidateUserWalletCache(ctx, updatedWallet.UserID)
	
	return updatedWallet, nil
}

// HasSufficientBalance checks if a wallet has enough balance
func (s *walletService) HasSufficientBalance(ctx context.Context, id string, amount float64) (bool, float64, error) {
	s.logger.Debug("Checking balance sufficiency", 
		zap.String("wallet_id", id), 
		zap.Float64("required_amount", amount),
	)
	
	// Try to get the wallet from cache first
	wallet, err := s.getWalletFromCache(ctx, id)
	if err == nil {
		return wallet.Balance >= amount, wallet.Balance, nil
	}
	
	// Cache miss, check in database
	isSufficient, balance, err := s.walletRepo.HasSufficientBalance(ctx, id, amount)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return false, 0, err
		}
		s.logger.Error("Failed to check balance sufficiency", 
			zap.Error(err), 
			zap.String("wallet_id", id),
		)
		return false, 0, fmt.Errorf("failed to check wallet balance: %w", err)
	}
	
	return isSufficient, balance, nil
}

// BlockWallet blocks a wallet
func (s *walletService) BlockWallet(ctx context.Context, id, reason string) (*model.Wallet, error) {
	s.logger.Info("Blocking wallet", zap.String("wallet_id", id), zap.String("reason", reason))
	
	blockedWallet, err := s.walletRepo.Block(ctx, id, reason)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, err
		}
		s.logger.Error("Failed to block wallet", zap.Error(err), zap.String("wallet_id", id))
		return nil, fmt.Errorf("failed to block wallet: %w", err)
	}
	
	// Update the cache
	if err := s.cacheWallet(ctx, blockedWallet); err != nil {
		s.logger.Warn("Failed to update wallet cache after blocking", zap.Error(err), zap.String("wallet_id", id))
	}
	
	return blockedWallet, nil
}

// UnblockWallet unblocks a wallet
func (s *walletService) UnblockWallet(ctx context.Context, id string) (*model.Wallet, error) {
	s.logger.Info("Unblocking wallet", zap.String("wallet_id", id))
	
	unblockedWallet, err := s.walletRepo.Unblock(ctx, id)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, err
		}
		s.logger.Error("Failed to unblock wallet", zap.Error(err), zap.String("wallet_id", id))
		return nil, fmt.Errorf("failed to unblock wallet: %w", err)
	}
	
	// Update the cache
	if err := s.cacheWallet(ctx, unblockedWallet); err != nil {
		s.logger.Warn("Failed to update wallet cache after unblocking", zap.Error(err), zap.String("wallet_id", id))
	}
	
	return unblockedWallet, nil
}

// *** Cache utility methods ***

// cacheWallet stores a wallet in the Redis cache
func (s *walletService) cacheWallet(ctx context.Context, wallet *model.Wallet) error {
	walletJSON, err := json.Marshal(wallet)
	if err != nil {
		return err
	}
	
	key := fmt.Sprintf("%s%s", walletCachePrefix, wallet.ID)
	if err := s.redisClient.Set(ctx, key, walletJSON, walletCacheTTL).Err(); err != nil {
		return err
	}
	
	return nil
}

// getWalletFromCache tries to retrieve a wallet from Redis cache
func (s *walletService) getWalletFromCache(ctx context.Context, id string) (*model.Wallet, error) {
	key := fmt.Sprintf("%s%s", walletCachePrefix, id)
	walletJSON, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	
	var wallet model.Wallet
	if err := json.Unmarshal([]byte(walletJSON), &wallet); err != nil {
		return nil, err
	}
	
	return &wallet, nil
}

// invalidateUserWalletCache clears any cached items related to a user's wallet
func (s *walletService) invalidateUserWalletCache(ctx context.Context, userID string) {
	// In a more complex implementation, we might maintain a mapping
	// between user IDs and wallet IDs in Redis
	s.logger.Debug("Cache invalidation would happen here for user wallet", zap.String("user_id", userID))
}