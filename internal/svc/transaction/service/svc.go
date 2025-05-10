package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/wallet/pkg/logger"
	"github.com/yourorg/wallet/transaction/clients"
	"github.com/yourorg/wallet/transaction/model"
	"github.com/yourorg/wallet/transaction/repository"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds for this transaction")
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInvalidAmount     = errors.New("invalid amount: must be greater than zero")
	ErrSameWallet        = errors.New("source and destination wallets cannot be the same")
	ErrTransactionFailed = errors.New("transaction failed")
)

// TransactionService defines the interface for transaction operations
type TransactionService interface {
	Deposit(ctx context.Context, walletID uuid.UUID, amount float64, description string) (*model.Transaction, error)
	Withdraw(ctx context.Context, walletID uuid.UUID, amount float64, description string) (*model.Transaction, error)
	Transfer(ctx context.Context, fromWalletID, toWalletID uuid.UUID, amount float64, description string) (*model.Transaction, error)
	GetTransaction(ctx context.Context, id uuid.UUID) (*model.Transaction, error)
	GetTransactionHistory(ctx context.Context, walletID uuid.UUID, filter model.TransactionFilter) (*model.PaginatedTransactions, error)
}

// TransactionServiceImpl implements the TransactionService interface
type TransactionServiceImpl struct {
	repo       repository.TransactionRepository
	walletSvc  clients.WalletServiceClient
	notifySvc  clients.NotificationServiceClient
	logger     logger.Logger
}

// NewTransactionService creates a new transaction service
func NewTransactionService(
	repo repository.TransactionRepository,
	walletSvc clients.WalletServiceClient,
	notifySvc clients.NotificationServiceClient,
	logger logger.Logger,
) TransactionService {
	return &TransactionServiceImpl{
		repo:      repo,
		walletSvc: walletSvc,
		notifySvc: notifySvc,
		logger:    logger,
	}
}

// Deposit adds funds to a wallet
func (s *TransactionServiceImpl) Deposit(
	ctx context.Context,
	walletID uuid.UUID,
	amount float64,
	description string,
) (*model.Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Check if wallet exists by getting balance
	_, err := s.walletSvc.GetBalance(ctx, walletID)
	if err != nil {
		s.logger.Error("Failed to get wallet balance", "error", err, "wallet_id", walletID)
		return nil, ErrWalletNotFound
	}

	// Begin database transaction
	dbTx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Use transaction in repository
	txRepo := s.repo.WithTx(dbTx)

	// Create transaction record
	tx := model.NewTransaction(
		model.Deposit,
		nil, // No source wallet for deposits
		&walletID,
		amount,
		description,
	)

	// Save transaction to database
	if err := txRepo.CreateTransaction(ctx, tx); err != nil {
		s.logger.Error("Failed to create transaction record", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Record initial event
	initialEvent := model.NewTransactionEvent(
		tx.ID,
		model.EventCreated,
		map[string]interface{}{
			"wallet_id": walletID.String(),
			"amount":    amount,
		},
	)
	if err := txRepo.AddTransactionEvent(ctx, initialEvent); err != nil {
		s.logger.Error("Failed to add transaction event", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to add transaction event: %w", err)
	}

	// Update wallet balance
	err = s.walletSvc.UpdateBalance(ctx, walletID, amount, false)
	if err != nil {
		s.logger.Error("Failed to update wallet balance", "error", err, "wallet_id", walletID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Mark transaction as completed
	if err := txRepo.UpdateTransactionStatus(ctx, tx.ID, model.Completed); err != nil {
		s.logger.Error("Failed to update transaction status", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Add completion event
	completionEvent := model.NewTransactionEvent(
		tx.ID,
		model.EventCompleted,
		map[string]interface{}{
			"wallet_id": walletID.String(),
			"amount":    amount,
		},
	)
	if err := txRepo.AddTransactionEvent(ctx, completionEvent); err != nil {
		s.logger.Error("Failed to add completion event", "error", err, "transaction_id", tx.ID)
		// Continue despite this error since the main transaction is successful
	}

	// Commit the database transaction
	if err := txRepo.CommitTx(dbTx); err != nil {
		s.logger.Error("Failed to commit transaction", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update transaction object
	tx.Status = model.Completed
	tx.UpdatedAt = time.Now()

	// Send notification asynchronously
	go func() {
		// Use a new context since the original might be canceled
		notifyCtx := context.Background()
		
		event := clients.NotificationEvent{
			Type:      clients.NotifyDeposit,
			WalletID:  walletID,
			Amount:    amount,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"transaction_id": tx.ID.String(),
				"description":    description,
			},
		}

		if err := s.notifySvc.PublishTransactionEvent(notifyCtx, event); err != nil {
			s.logger.Error("Failed to publish notification", "error", err, "transaction_id", tx.ID)
		}
	}()

	return tx, nil
}

// Withdraw removes funds from a wallet
func (s *TransactionServiceImpl) Withdraw(
	ctx context.Context,
	walletID uuid.UUID,
	amount float64,
	description string,
) (*model.Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Check wallet balance
	balance, err := s.walletSvc.GetBalance(ctx, walletID)
	if err != nil {
		s.logger.Error("Failed to get wallet balance", "error", err, "wallet_id", walletID)
		return nil, ErrWalletNotFound
	}

	// Check if there are sufficient funds
	if balance < amount {
		return nil, ErrInsufficientFunds
	}

	// Begin database transaction
	dbTx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Use transaction in repository
	txRepo := s.repo.WithTx(dbTx)

	// Create transaction record
	tx := model.NewTransaction(
		model.Withdraw,
		&walletID,
		nil, // No destination wallet for withdrawals
		amount,
		description,
	)

	// Save transaction to database
	if err := txRepo.CreateTransaction(ctx, tx); err != nil {
		s.logger.Error("Failed to create transaction record", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Record initial event
	initialEvent := model.NewTransactionEvent(
		tx.ID,
		model.EventCreated,
		map[string]interface{}{
			"wallet_id": walletID.String(),
			"amount":    amount,
		},
	)
	if err := txRepo.AddTransactionEvent(ctx, initialEvent); err != nil {
		s.logger.Error("Failed to add transaction event", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to add transaction event: %w", err)
	}

	// Update wallet balance
	err = s.walletSvc.UpdateBalance(ctx, walletID, amount, true)
	if err != nil {
		s.logger.Error("Failed to update wallet balance", "error", err, "wallet_id", walletID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to update wallet balance: %w", err)
	}

	// Mark transaction as completed
	if err := txRepo.UpdateTransactionStatus(ctx, tx.ID, model.Completed); err != nil {
		s.logger.Error("Failed to update transaction status", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Add completion event
	completionEvent := model.NewTransactionEvent(
		tx.ID,
		model.EventCompleted,
		map[string]interface{}{
			"wallet_id": walletID.String(),
			"amount":    amount,
		},
	)
	if err := txRepo.AddTransactionEvent(ctx, completionEvent); err != nil {
		s.logger.Error("Failed to add completion event", "error", err, "transaction_id", tx.ID)
		// Continue despite this error since the main transaction is successful
	}

	// Commit the database transaction
	if err := txRepo.CommitTx(dbTx); err != nil {
		s.logger.Error("Failed to commit transaction", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update transaction object
	tx.Status = model.Completed
	tx.UpdatedAt = time.Now()

	// Send notification asynchronously
	go func() {
		// Use a new context since the original might be canceled
		notifyCtx := context.Background()
		
		event := clients.NotificationEvent{
			Type:      clients.NotifyWithdrawal,
			WalletID:  walletID,
			Amount:    amount,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"transaction_id": tx.ID.String(),
				"description":    description,
			},
		}

		if err := s.notifySvc.PublishTransactionEvent(notifyCtx, event); err != nil {
			s.logger.Error("Failed to publish notification", "error", err, "transaction_id", tx.ID)
		}
	}()

	return tx, nil
}

// Transfer moves funds between wallets
func (s *TransactionServiceImpl) Transfer(
	ctx context.Context,
	fromWalletID, toWalletID uuid.UUID,
	amount float64,
	description string,
) (*model.Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	if fromWalletID == toWalletID {
		return nil, ErrSameWallet
	}

	// Check if source wallet exists and has sufficient funds
	fromBalance, err := s.walletSvc.GetBalance(ctx, fromWalletID)
	if err != nil {
		s.logger.Error("Failed to get source wallet balance", "error", err, "wallet_id", fromWalletID)
		return nil, ErrWalletNotFound
	}
	
	if fromBalance < amount {
		return nil, ErrInsufficientFunds
	}

	// Check if destination wallet exists
	_, err = s.walletSvc.GetBalance(ctx, toWalletID)
	if err != nil {
		s.logger.Error("Failed to get destination wallet balance", "error", err, "wallet_id", toWalletID)
		return nil, ErrWalletNotFound
	}

	// Begin database transaction
	dbTx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	// Use transaction in repository
	txRepo := s.repo.WithTx(dbTx)

	// Create transaction record
	tx := model.NewTransaction(
		model.Transfer,
		&fromWalletID,
		&toWalletID,
		amount,
		description,
	)

	// Save transaction to database
	if err := txRepo.CreateTransaction(ctx, tx); err != nil {
		s.logger.Error("Failed to create transaction record", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Record initial event
	initialEvent := model.NewTransactionEvent(
		tx.ID,
		model.EventCreated,
		map[string]interface{}{
			"from_wallet_id": fromWalletID.String(),
			"to_wallet_id":   toWalletID.String(),
			"amount":         amount,
		},
	)
	if err := txRepo.AddTransactionEvent(ctx, initialEvent); err != nil {
		s.logger.Error("Failed to add transaction event", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to add transaction event: %w", err)
	}

	// Deduct from source wallet
	err = s.walletSvc.UpdateBalance(ctx, fromWalletID, amount, true)
	if err != nil {
		s.logger.Error("Failed to update source wallet balance", "error", err, "wallet_id", fromWalletID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to update source wallet balance: %w", err)
	}

	// Add to destination wallet
	err = s.walletSvc.UpdateBalance(ctx, toWalletID, amount, false)
	if err != nil {
		s.logger.Error("Failed to update destination wallet balance", "error", err, "wallet_id", toWalletID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to update destination wallet balance: %w", err)
	}

	// Mark transaction as completed
	if err := txRepo.UpdateTransactionStatus(ctx, tx.ID, model.Completed); err != nil {
		s.logger.Error("Failed to update transaction status", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Add completion event
	completionEvent := model.NewTransactionEvent(
		tx.ID,
		model.EventCompleted,
		map[string]interface{}{
			"from_wallet_id": fromWalletID.String(),
			"to_wallet_id":   toWalletID.String(),
			"amount":         amount,
		},
	)
	if err := txRepo.AddTransactionEvent(ctx, completionEvent); err != nil {
		s.logger.Error("Failed to add completion event", "error", err, "transaction_id", tx.ID)
		// Continue despite this error since the main transaction is successful
	}

	// Commit the database transaction
	if err := txRepo.CommitTx(dbTx); err != nil {
		s.logger.Error("Failed to commit transaction", "error", err, "transaction_id", tx.ID)
		txRepo.RollbackTx(dbTx)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update transaction object
	tx.Status = model.Completed
	tx.UpdatedAt = time.Now()

	// Send notifications asynchronously
	go func() {
		// Use a new context since the original might be canceled
		notifyCtx := context.Background()
		
		// Notification for sender
		senderEvent := clients.NotificationEvent{
			Type:      clients.NotifyTransferSent,
			WalletID:  fromWalletID,
			Amount:    amount,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"transaction_id":      tx.ID.String(),
				"destination_wallet":   toWalletID.String(),
				"description":         description,
			},
		}

		if err := s.notifySvc.PublishTransactionEvent(notifyCtx, senderEvent); err != nil {
			s.logger.Error("Failed to publish sender notification", "error", err, "transaction_id", tx.ID)
		}

		// Notification for recipient
		recipientEvent := clients.NotificationEvent{
			Type:      clients.NotifyTransferReceived,
			WalletID:  toWalletID,
			Amount:    amount,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"transaction_id": tx.ID.String(),
				"source_wallet":  fromWalletID.String(),
				"description":    description,
			},
		}

		if err := s.notifySvc.PublishTransactionEvent(notifyCtx, recipientEvent); err != nil {
			s.logger.Error("Failed to publish recipient notification", "error", err, "transaction_id", tx.ID)
		}
	}()

	return tx, nil
}

// GetTransaction retrieves a transaction by ID
func (s *TransactionServiceImpl) GetTransaction(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	return s.repo.GetTransactionByID(ctx, id)
}

// GetTransactionHistory returns a paginated list of transactions for a wallet
func (s *TransactionServiceImpl) GetTransactionHistory(
	ctx context.Context,
	walletID uuid.UUID,
	filter model.TransactionFilter,
) (*model.PaginatedTransactions, error) {
	return s.repo.GetTransactionsForWallet(ctx, walletID, filter)
}