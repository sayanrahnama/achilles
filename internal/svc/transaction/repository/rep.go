package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/yourorg/wallet/pkg/postgres"
	"github.com/yourorg/wallet/transaction/model"
)

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	CreateTransaction(ctx context.Context, tx *model.Transaction) error
	UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status model.TransactionStatus) error
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error)
	GetTransactionsForWallet(ctx context.Context, walletID uuid.UUID, filter model.TransactionFilter) (*model.PaginatedTransactions, error)
	AddTransactionEvent(ctx context.Context, event *model.TransactionEvent) error
	GetTransactionEvents(ctx context.Context, transactionID uuid.UUID) ([]model.TransactionEvent, error)
	WithTx(tx *sql.Tx) TransactionRepository
	// Database transaction operations
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CommitTx(tx *sql.Tx) error
	RollbackTx(tx *sql.Tx) error
}

// PostgresTransactionRepository implements TransactionRepository interface
type PostgresTransactionRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *sql.DB) TransactionRepository {
	return &PostgresTransactionRepository{
		db: db,
	}
}

// WithTx creates a new repository with the given transaction
func (r *PostgresTransactionRepository) WithTx(tx *sql.Tx) TransactionRepository {
	return &PostgresTransactionRepository{
		db: r.db,
		tx: tx,
	}
}

// getDB returns either the transaction or database connection
func (r *PostgresTransactionRepository) getDB() postgres.DBTX {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// BeginTx starts a new database transaction
func (r *PostgresTransactionRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
}

// CommitTx commits a database transaction
func (r *PostgresTransactionRepository) CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

// RollbackTx rolls back a database transaction
func (r *PostgresTransactionRepository) RollbackTx(tx *sql.Tx) error {
	return tx.Rollback()
}

// CreateTransaction inserts a new transaction record in the database
func (r *PostgresTransactionRepository) CreateTransaction(ctx context.Context, tx *model.Transaction) error {
	query := `
		INSERT INTO transactions (
			id, transaction_type, source_wallet_id, destination_wallet_id, 
			amount, status, description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.getDB().ExecContext(
		ctx,
		query,
		tx.ID,
		tx.TransactionType,
		tx.SourceWalletID,
		tx.DestinationWalletID,
		tx.Amount,
		tx.Status,
		tx.Description,
		tx.CreatedAt,
		tx.UpdatedAt,
	)

	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			// Log specific PostgreSQL errors
			return fmt.Errorf("database error creating transaction: %s (Code: %s)", pqErr.Message, pqErr.Code)
		}
		return fmt.Errorf("error creating transaction: %w", err)
	}

	return nil
}

// UpdateTransactionStatus updates the status of a transaction
func (r *PostgresTransactionRepository) UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status model.TransactionStatus) error {
	query := `
		UPDATE transactions 
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.getDB().ExecContext(
		ctx,
		query,
		status,
		time.Now(),
		id,
	)

	if err != nil {
		return fmt.Errorf("error updating transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("transaction not found")
	}

	return nil
}

// GetTransactionByID retrieves a transaction by its ID
func (r *PostgresTransactionRepository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	query := `
		SELECT 
			id, transaction_type, source_wallet_id, destination_wallet_id,
			amount, status, description, created_at, updated_at
		FROM transactions
		WHERE id = $1
	`

	var tx model.Transaction
	var sourceID, destID sql.NullString

	err := r.getDB().QueryRowContext(ctx, query, id).Scan(
		&tx.ID,
		&tx.TransactionType,
		&sourceID,
		&destID,
		&tx.Amount,
		&tx.Status,
		&tx.Description,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("transaction not found")
		}
		return nil, fmt.Errorf("error getting transaction: %w", err)
	}

	// Convert sql.NullString to *uuid.UUID
	if sourceID.Valid {
		sourceUUID, err := uuid.Parse(sourceID.String)
		if err == nil {
			tx.SourceWalletID = &sourceUUID
		}
	}

	if destID.Valid {
		destUUID, err := uuid.Parse(destID.String)
		if err == nil {
			tx.DestinationWalletID = &destUUID
		}
	}

	return &tx, nil
}

// GetTransactionsForWallet returns transactions related to a wallet with pagination
func (r *PostgresTransactionRepository) GetTransactionsForWallet(
	ctx context.Context,
	walletID uuid.UUID,
	filter model.TransactionFilter,
) (*model.PaginatedTransactions, error) {
	// Default values for pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	// Build the base query
	baseQuery := `
		FROM transactions
		WHERE (source_wallet_id = $1 OR destination_wallet_id = $1)
	`

	// Apply filters
	params := []interface{}{walletID}
	paramCount := 1

	if filter.Type != nil {
		paramCount++
		baseQuery += fmt.Sprintf(" AND transaction_type = $%d", paramCount)
		params = append(params, *filter.Type)
	}

	if filter.Status != nil {
		paramCount++
		baseQuery += fmt.Sprintf(" AND status = $%d", paramCount)
		params = append(params, *filter.Status)
	}

	if filter.StartDate != nil {
		paramCount++
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", paramCount)
		params = append(params, *filter.StartDate)
	}

	if filter.EndDate != nil {
		paramCount++
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", paramCount)
		params = append(params, *filter.EndDate)
	}

	// Count query for pagination
	countQuery := "SELECT COUNT(*) " + baseQuery
	var totalCount int
	err := r.getDB().QueryRowContext(ctx, countQuery, params...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("error counting transactions: %w", err)
	}

	// Calculate total pages
	totalPages := (totalCount + filter.Limit - 1) / filter.Limit

	// Determine sort order
	sortField := "created_at"
	if filter.SortBy != "" {
		// Validate sort field to prevent SQL injection
		validFields := map[string]bool{
			"created_at":       true,
			"amount":           true,
			"transaction_type": true,
			"status":           true,
		}
		if validFields[filter.SortBy] {
			sortField = filter.SortBy
		}
	}

	sortDirection := "DESC"
	if filter.SortAscending {
		sortDirection = "ASC"
	}

	// Build the final query with pagination
	query := fmt.Sprintf(`
		SELECT 
			id, transaction_type, source_wallet_id, destination_wallet_id,
			amount, status, description, created_at, updated_at
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, baseQuery, sortField, sortDirection, paramCount+1, paramCount+2)

	// Add pagination parameters
	params = append(params, filter.Limit, (filter.Page-1)*filter.Limit)

	// Execute the query
	rows, err := r.getDB().QueryContext(ctx, query, params...)
	if err != nil {
		return nil, fmt.Errorf("error querying transactions: %w", err)
	}
	defer rows.Close()

	// Parse the results
	var transactions []model.Transaction
	for rows.Next() {
		var tx model.Transaction
		var sourceID, destID sql.NullString

		err := rows.Scan(
			&tx.ID,
			&tx.TransactionType,
			&sourceID,
			&destID,
			&tx.Amount,
			&tx.Status,
			&tx.Description,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning transaction row: %w", err)
		}

		// Convert sql.NullString to *uuid.UUID
		if sourceID.Valid {
			sourceUUID, err := uuid.Parse(sourceID.String)
			if err == nil {
				tx.SourceWalletID = &sourceUUID
			}
		}

		if destID.Valid {
			destUUID, err := uuid.Parse(destID.String)
			if err == nil {
				tx.DestinationWalletID = &destUUID
			}
		}

		transactions = append(transactions, tx)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	return &model.PaginatedTransactions{
		Transactions: transactions,
		TotalCount:   totalCount,
		Page:         filter.Page,
		TotalPages:   totalPages,
	}, nil
}

// AddTransactionEvent adds a new event to a transaction's lifecycle
func (r *PostgresTransactionRepository) AddTransactionEvent(ctx context.Context, event *model.TransactionEvent) error {
	query := `
		INSERT INTO transaction_events (
			id, transaction_id, event_type, details, created_at
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.getDB().ExecContext(
		ctx,
		query,
		event.ID,
		event.TransactionID,
		event.EventType,
		event.Details,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("error creating transaction event: %w", err)
	}

	return nil
}

// GetTransactionEvents retrieves all events for a transaction
func (r *PostgresTransactionRepository) GetTransactionEvents(ctx context.Context, transactionID uuid.UUID) ([]model.TransactionEvent, error) {
	query := `
		SELECT 
			id, transaction_id, event_type, details, created_at
		FROM transaction_events
		WHERE transaction_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.getDB().QueryContext(ctx, query, transactionID)
	if err != nil {
		return nil, fmt.Errorf("error querying transaction events: %w", err)
	}
	defer rows.Close()

	var events []model.TransactionEvent
	for rows.Next() {
		var event model.TransactionEvent
		var detailsJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.TransactionID,
			&event.EventType,
			&detailsJSON,
			&event.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning event row: %w", err)
		}

		// Parse the JSON details
		// This is simplified; in a real implementation you might use a JSON library
		event.Details = make(map[string]interface{})
		// Parse JSON detailsJSON into event.Details

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating event rows: %w", err)
	}

	return events, nil
}