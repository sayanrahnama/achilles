package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// WalletRepository defines the interface for wallet data operations
type WalletRepository interface {
	Create(ctx context.Context, wallet *model.Wallet) (*model.Wallet, error)
	GetByID(ctx context.Context, id string) (*model.Wallet, error)
	GetByUserID(ctx context.Context, userID string) (*model.Wallet, error)
	UpdateBalance(ctx context.Context, id string, amount float64) (*model.Wallet, error)
	HasSufficientBalance(ctx context.Context, id string, amount float64) (bool, float64, error)
	Block(ctx context.Context, id string, reason string) (*model.Wallet, error)
	Unblock(ctx context.Context, id string) (*model.Wallet, error)
}

type walletRepository struct {
	db *sql.DB
}

// NewWalletRepository creates a new instance of WalletRepository
func NewWalletRepository(db *sql.DB) WalletRepository {
	return &walletRepository{
		db: db,
	}
}

// Create inserts a new wallet into the database
func (r *walletRepository) Create(ctx context.Context, wallet *model.Wallet) (*model.Wallet, error) {
	query := `
		INSERT INTO wallets (user_id, balance)
		VALUES ($1, $2)
		RETURNING id, user_id, balance, is_blocked, block_reason, created_at, updated_at
	`

	row := r.db.QueryRowContext(ctx, query, wallet.UserID, wallet.Balance)
	
	var result model.Wallet
	err := row.Scan(
		&result.ID,
		&result.UserID,
		&result.Balance,
		&result.IsBlocked,
		&result.BlockReason,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetByID fetches a wallet by its ID
func (r *walletRepository) GetByID(ctx context.Context, id string) (*model.Wallet, error) {
	query := `
		SELECT id, user_id, balance, is_blocked, block_reason, created_at, updated_at
		FROM wallets
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	
	var wallet model.Wallet
	var blockReason sql.NullString
	
	err := row.Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.IsBlocked,
		&blockReason,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	
	if blockReason.Valid {
		wallet.BlockReason = blockReason.String
	}

	return &wallet, nil
}

// GetByUserID fetches a wallet by its owner's user ID
func (r *walletRepository) GetByUserID(ctx context.Context, userID string) (*model.Wallet, error) {
	query := `
		SELECT id, user_id, balance, is_blocked, block_reason, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
	`

	row := r.db.QueryRowContext(ctx, query, userID)
	
	var wallet model.Wallet
	var blockReason sql.NullString
	
	err := row.Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.IsBlocked,
		&blockReason,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	
	if blockReason.Valid {
		wallet.BlockReason = blockReason.String
	}

	return &wallet, nil
}

// UpdateBalance updates a wallet's balance using a transaction to ensure data integrity
func (r *walletRepository) UpdateBalance(ctx context.Context, id string, amount float64) (*model.Wallet, error) {
	// Start a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	
	// Ensure transaction is rolled back in case of error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// First check if wallet exists and if balance is sufficient
	checkQuery := `
		SELECT balance
		FROM wallets
		WHERE id = $1
		FOR UPDATE
	`
	
	var currentBalance float64
	err = tx.QueryRowContext(ctx, checkQuery, id).Scan(&currentBalance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	
	// Check for negative balance if withdrawing
	newBalance := currentBalance + amount
	if newBalance < 0 {
		return nil, ErrInsufficientFunds
	}
	
	// Update the balance
	updateQuery := `
		UPDATE wallets
		SET balance = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, user_id, balance, is_blocked, block_reason, created_at, updated_at
	`
	
	var wallet model.Wallet
	var blockReason sql.NullString
	
	err = tx.QueryRowContext(
		ctx,
		updateQuery,
		newBalance,
		time.Now(),
		id,
	).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.IsBlocked,
		&blockReason,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	if blockReason.Valid {
		wallet.BlockReason = blockReason.String
	}
	
	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	
	return &wallet, nil
}

// HasSufficientBalance checks if a wallet has enough balance for a transaction
func (r *walletRepository) HasSufficientBalance(ctx context.Context, id string, amount float64) (bool, float64, error) {
	query := `
		SELECT balance
		FROM wallets
		WHERE id = $1
	`
	
	var balance float64
	err := r.db.QueryRowContext(ctx, query, id).Scan(&balance)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, ErrWalletNotFound
		}
		return false, 0, err
	}
	
	return balance >= amount, balance, nil
}

// Block marks a wallet as blocked
func (r *walletRepository) Block(ctx context.Context, id string, reason string) (*model.Wallet, error) {
	query := `
		UPDATE wallets
		SET is_blocked = true, block_reason = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, user_id, balance, is_blocked, block_reason, created_at, updated_at
	`
	
	var wallet model.Wallet
	var blockReason sql.NullString
	
	err := r.db.QueryRowContext(
		ctx,
		query,
		reason,
		time.Now(),
		id,
	).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.IsBlocked,
		&blockReason,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	
	if blockReason.Valid {
		wallet.BlockReason = blockReason.String
	}
	
	return &wallet, nil
}

// Unblock marks a wallet as unblocked
func (r *walletRepository) Unblock(ctx context.Context, id string) (*model.Wallet, error) {
	query := `
		UPDATE wallets
		SET is_blocked = false, block_reason = NULL, updated_at = $1
		WHERE id = $2
		RETURNING id, user_id, balance, is_blocked, block_reason, created_at, updated_at
	`
	
	var wallet model.Wallet
	var blockReason sql.NullString
	
	err := r.db.QueryRowContext(
		ctx,
		query,
		time.Now(),
		id,
	).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Balance,
		&wallet.IsBlocked,
		&blockReason,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	
	if blockReason.Valid {
		wallet.BlockReason = blockReason.String
	}
	
	return &wallet, nil
}