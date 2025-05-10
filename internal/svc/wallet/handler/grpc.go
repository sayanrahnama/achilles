// wallet/handler/grpc.go
package handler

import (
	"context"
	"time"
	"wallet/model"
	"wallet/pb/wallet"
	"wallet/repository"
	"wallet/service"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type WalletHandler struct {
	wallet.UnimplementedWalletServiceServer
	logger  *zap.Logger
	service service.WalletService
}

// NewWalletHandler creates a new gRPC handler for wallet service
func NewWalletHandler(logger *zap.Logger, service service.WalletService) *WalletHandler {
	return &WalletHandler{
		logger:  logger,
		service: service,
	}
}

// CreateWallet handles wallet creation requests
func (h *WalletHandler) CreateWallet(ctx context.Context, req *wallet.CreateWalletRequest) (*wallet.Wallet, error) {
	h.logger.Info("Handling CreateWallet request", zap.String("user_id", req.UserId))
	
	result, err := h.service.CreateWallet(ctx, req.UserId, req.InitialBalance)
	if err != nil {
		h.logger.Error("Failed to create wallet", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to create wallet: %v", err)
	}
	
	return h.mapModelToProto(result), nil
}

// GetWallet handles requests to fetch a wallet by ID
func (h *WalletHandler) GetWallet(ctx context.Context, req *wallet.GetWalletRequest) (*wallet.Wallet, error) {
	h.logger.Info("Handling GetWallet request", zap.String("wallet_id", req.WalletId))
	
	result, err := h.service.GetWallet(ctx, req.WalletId)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found")
		}
		h.logger.Error("Failed to get wallet", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get wallet: %v", err)
	}
	
	return h.mapModelToProto(result), nil
}

// GetWalletByUserId handles requests to fetch a wallet by user ID
func (h *WalletHandler) GetWalletByUserId(ctx context.Context, req *wallet.GetWalletByUserIdRequest) (*wallet.Wallet, error) {
	h.logger.Info("Handling GetWalletByUserId request", zap.String("user_id", req.UserId))
	
	result, err := h.service.GetWalletByUserID(ctx, req.UserId)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found for user")
		}
		h.logger.Error("Failed to get wallet by user ID", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to get wallet: %v", err)
	}
	
	return h.mapModelToProto(result), nil
}

// UpdateBalance handles requests to update a wallet's balance
func (h *WalletHandler) UpdateBalance(ctx context.Context, req *wallet.UpdateBalanceRequest) (*wallet.Wallet, error) {
	h.logger.Info("Handling UpdateBalance request", 
		zap.String("wallet_id", req.WalletId),
		zap.Float64("amount", req.Amount),
	)
	
	result, err := h.service.UpdateBalance(ctx, req.WalletId, req.Amount)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found")
		}
		if err == repository.ErrInsufficientFunds {
			return nil, status.Error(codes.FailedPrecondition, "insufficient funds")
		}
		h.logger.Error("Failed to update wallet balance", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update wallet balance: %v", err)
	}
	
	return h.mapModelToProto(result), nil
}

// HasSufficientBalance checks if a wallet has enough funds
func (h *WalletHandler) HasSufficientBalance(ctx context.Context, req *wallet.HasSufficientBalanceRequest) (*wallet.HasSufficientBalanceResponse, error) {
	h.logger.Info("Handling HasSufficientBalance request", 
		zap.String("wallet_id", req.WalletId),
		zap.Float64("amount", req.Amount),
	)
	
	isSufficient, balance, err := h.service.HasSufficientBalance(ctx, req.WalletId, req.Amount)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found")
		}
		h.logger.Error("Failed to check balance sufficiency", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to check wallet balance: %v", err)
	}
	
	return &wallet.HasSufficientBalanceResponse{
		IsSufficient:    isSufficient,
		CurrentBalance:  balance,
	}, nil
}

// BlockWallet handles requests to block a wallet
func (h *WalletHandler) BlockWallet(ctx context.Context, req *wallet.BlockWalletRequest) (*wallet.Wallet, error) {
	h.logger.Info("Handling BlockWallet request", 
		zap.String("wallet_id", req.WalletId),
		zap.String("reason", req.Reason),
	)
	
	result, err := h.service.BlockWallet(ctx, req.WalletId, req.Reason)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found")
		}
		h.logger.Error("Failed to block wallet", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to block wallet: %v", err)
	}
	
	return h.mapModelToProto(result), nil
}

// UnblockWallet handles requests to unblock a wallet
func (h *WalletHandler) UnblockWallet(ctx context.Context, req *wallet.UnblockWalletRequest) (*wallet.Wallet, error) {
	h.logger.Info("Handling UnblockWallet request", zap.String("wallet_id", req.WalletId))
	
	result, err := h.service.UnblockWallet(ctx, req.WalletId)
	if err != nil {
		if err == repository.ErrWalletNotFound {
			return nil, status.Error(codes.NotFound, "wallet not found")
		}
		h.logger.Error("Failed to unblock wallet", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to unblock wallet: %v", err)
	}
	
	return h.mapModelToProto(result), nil
}

// mapModelToProto converts a wallet model to protobuf response
func (h *WalletHandler) mapModelToProto(wallet *model.Wallet) *wallet.Wallet {
	createdAt := timestamppb.New(wallet.CreatedAt)
	updatedAt := timestamppb.New(wallet.UpdatedAt)
	
	return &wallet.Wallet{
		Id:          wallet.ID,
		UserId:      wallet.UserID,
		Balance:     wallet.Balance,
		IsBlocked:   wallet.IsBlocked,
		BlockReason: wallet.BlockReason,
		CreatedAt:   createdAt.AsTime().Format(time.RFC3339),
		UpdatedAt:   updatedAt.AsTime().Format(time.RFC3339),
	}
}