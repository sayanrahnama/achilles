package handler

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/yourorg/wallet/transaction/pb/transaction"
	"github.com/yourorg/wallet/pkg/logger"
	"github.com/yourorg/wallet/transaction/model"
	"github.com/yourorg/wallet/transaction/service"
)

// GRPCHandler implements the gRPC service handler for transactions
type GRPCHandler struct {
	pb.UnimplementedTransactionServiceServer
	service service.TransactionService
	logger  logger.Logger
}

// NewGRPCHandler creates a new gRPC handler
func NewGRPCHandler(svc service.TransactionService, logger logger.Logger) *GRPCHandler {
	return &GRPCHandler{
		service: svc,
		logger:  logger,
	}
}

// Deposit handles deposit requests
func (h *GRPCHandler) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.TransactionResponse, error) {
	walletID, err := uuid.Parse(req.WalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet ID: %v", err)
	}

	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than zero")
	}

	tx, err := h.service.Deposit(ctx, walletID, req.Amount, req.Description)
	if err != nil {
		h.logger.Error("Failed to process deposit", "error", err, "wallet_id", walletID)
		
		var statusCode codes.Code
		var statusMsg string
		
		switch {
		case errors.Is(err, service.ErrWalletNotFound):
			statusCode = codes.NotFound
			statusMsg = "wallet not found"
		case errors.Is(err, service.ErrInvalidAmount):
			statusCode = codes.InvalidArgument
			statusMsg = "invalid amount"
		default:
			statusCode = codes.Internal
			statusMsg = "failed to process deposit"
		}
		
		return nil, status.Errorf(statusCode, statusMsg)
	}

	return &pb.TransactionResponse{
		TransactionId: tx.ID.String(),
		Status:        200,
		Message:       "Deposit processed successfully",
		Transaction:   convertTransactionToProto(tx),
	}, nil
}

// Withdraw handles withdrawal requests
func (h *GRPCHandler) Withdraw(ctx context.Context, req *pb.WithdrawRequest) (*pb.TransactionResponse, error) {
	walletID, err := uuid.Parse(req.WalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet ID: %v", err)
	}

	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than zero")
	}

	tx, err := h.service.Withdraw(ctx, walletID, req.Amount, req.Description)
	if err != nil {
		h.logger.Error("Failed to process withdrawal", "error", err, "wallet_id", walletID)
		
		var statusCode codes.Code
		var statusMsg string
		
		switch {
		case errors.Is(err, service.ErrWalletNotFound):
			statusCode = codes.NotFound
			statusMsg = "wallet not found"
		case errors.Is(err, service.ErrInsufficientFunds):
			statusCode = codes.FailedPrecondition
			statusMsg = "insufficient funds"
		case errors.Is(err, service.ErrInvalidAmount):
			statusCode = codes.InvalidArgument
			statusMsg = "invalid amount"
		default:
			statusCode = codes.Internal
			statusMsg = "failed to process withdrawal"
		}
		
		return nil, status.Errorf(statusCode, statusMsg)
	}

	return &pb.TransactionResponse{
		TransactionId: tx.ID.String(),
		Status:        200,
		Message:       "Withdrawal processed successfully",
		Transaction:   convertTransactionToProto(tx),
	}, nil
}

// Transfer handles transfer requests between wallets
func (h *GRPCHandler) Transfer(ctx context.Context, req *pb.TransferRequest) (*pb.TransactionResponse, error) {
	fromWalletID, err := uuid.Parse(req.FromWalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source wallet ID: %v", err)
	}

	toWalletID, err := uuid.Parse(req.ToWalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid destination wallet ID: %v", err)
	}

	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be greater than zero")
	}

	if fromWalletID == toWalletID {
		return nil, status.Error(codes.InvalidArgument, "source and destination wallets cannot be the same")
	}

	tx, err := h.service.Transfer(ctx, fromWalletID, toWalletID, req.Amount, req.Description)
	if err != nil {
		h.logger.Error("Failed to process transfer", 
			"error", err, 
			"from_wallet_id", fromWalletID,
			"to_wallet_id", toWalletID,
		)
		
		var statusCode codes.Code
		var statusMsg string
		
		switch {
		case errors.Is(err, service.ErrWalletNotFound):
			statusCode = codes.NotFound
			statusMsg = "wallet not found"
		case errors.Is(err, service.ErrInsufficientFunds):
			statusCode = codes.FailedPrecondition
			statusMsg = "insufficient funds"
		case errors.Is(err, service.ErrInvalidAmount):
			statusCode = codes.InvalidArgument
			statusMsg = "invalid amount"
		case errors.Is(err, service.ErrSameWallet):
			statusCode = codes.InvalidArgument
			statusMsg = "source and destination wallets cannot be the same"
		default:
			statusCode = codes.Internal
			statusMsg = "failed to process transfer"
		}
		
		return nil, status.Errorf(statusCode, statusMsg)
	}

	return &pb.TransactionResponse{
		TransactionId: tx.ID.String(),
		Status:        200,
		Message:       "Transfer processed successfully",
		Transaction:   convertTransactionToProto(tx),
	}, nil
}

// GetTransactionHistory handles requests for a wallet's transaction history
func (h *GRPCHandler) GetTransactionHistory(
	ctx context.Context, 
	req *pb.TransactionHistoryRequest,
) (*pb.TransactionHistoryResponse, error) {
	walletID, err := uuid.Parse(req.WalletId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet ID: %v", err)
	}

	// Set up filter
	filter := model.TransactionFilter{
		Page:          int(req.Page),
		Limit:         int(req.Limit),
		SortBy:        req.SortBy,
		SortAscending: req.Ascending,
	}

	// Apply filter
	result, err := h.service.GetTransactionHistory(ctx, walletID, filter)
	if err != nil {
		h.logger.Error("Failed to get transaction history", "error", err, "wallet_id", walletID)
		return nil, status.Errorf(codes.Internal, "failed to get transaction history: %v", err)
	}

	// Convert model transactions to proto transactions
	protoTransactions := make([]*pb.Transaction, 0, len(result.Transactions))
	for _, tx := range result.Transactions {
		protoTransactions = append(protoTransactions, convertTransactionToProto(&tx))
	}

	return &pb.TransactionHistoryResponse{
		Transactions: protoTransactions,
		TotalCount:   int32(result.TotalCount),
		Page:         int32(result.Page),
		TotalPages:   int32(result.TotalPages),
	}, nil
}

// GetTransaction handles requests for a specific transaction by ID
func (h *GRPCHandler) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.Transaction, error) {
	transactionID, err := uuid.Parse(req.TransactionId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid transaction ID: %v", err)
	}

	tx, err := h.service.GetTransaction(ctx, transactionID)
	if err != nil {
		h.logger.Error("Failed to get transaction", "error", err, "transaction_id", transactionID)
		
		// Determine error type
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "transaction not found" {
			return nil, status.Error(codes.NotFound, "transaction not found")
		}
		
		return nil, status.Errorf(codes.Internal, "failed to get transaction: %v", err)
	}

	return convertTransactionToProto(tx), nil
}

// convertTransactionToProto converts a model.Transaction to pb.Transaction
func convertTransactionToProto(tx *model.Transaction) *pb.Transaction {
	protoTx := &pb.Transaction{
		Id:             tx.ID.String(),
		TransactionType: string(tx.TransactionType),
		Amount:         tx.Amount,
		Status:         string(tx.Status),
		Description:    tx.Description,
		Timestamp:      tx.CreatedAt.Unix(),
	}

	// Add source wallet ID if present
	if tx.SourceWalletID != nil {
		protoTx.SourceWalletId = tx.SourceWalletID.String()
	}

	// Add destination wallet ID if present
	if tx.DestinationWalletID != nil {
		protoTx.DestinationWalletId = tx.DestinationWalletID.String()
	}

	return protoTx
}