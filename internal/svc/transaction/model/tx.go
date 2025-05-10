package model

import (
	"time"

	"github.com/google/uuid"
)

// TransactionType represents the type of transaction
type TransactionType string

// TransactionStatus represents the status of a transaction
type TransactionStatus string

// Event types for transaction events
type EventType string

const (
	// Transaction types
	Deposit  TransactionType = "deposit"
	Withdraw TransactionType = "withdraw"
	Transfer TransactionType = "transfer"

	// Transaction statuses
	Pending   TransactionStatus = "pending"
	Completed TransactionStatus = "completed"
	Failed    TransactionStatus = "failed"

	// Event types
	EventCreated          EventType = "created"
	EventProcessing       EventType = "processing"
	EventCompleted        EventType = "completed"
	EventFailed           EventType = "failed"
	EventNotificationSent EventType = "notification_sent"
)

// Transaction represents a financial transaction in the system
type Transaction struct {
	ID                 uuid.UUID         `json:"id"`
	TransactionType    TransactionType   `json:"transaction_type"`
	SourceWalletID     *uuid.UUID        `json:"source_wallet_id"`
	DestinationWalletID *uuid.UUID       `json:"destination_wallet_id"`
	Amount             float64           `json:"amount"`
	Status             TransactionStatus `json:"status"`
	Description        string            `json:"description"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// TransactionEvent represents an event in the lifecycle of a transaction
type TransactionEvent struct {
	ID            uuid.UUID          `json:"id"`
	TransactionID uuid.UUID          `json:"transaction_id"`
	EventType     EventType          `json:"event_type"`
	Details       map[string]interface{} `json:"details"`
	CreatedAt     time.Time          `json:"created_at"`
}

// TransactionFilter contains parameters for filtering transactions
type TransactionFilter struct {
	WalletID    *uuid.UUID
	Type        *TransactionType
	Status      *TransactionStatus
	StartDate   *time.Time
	EndDate     *time.Time
	Page        int
	Limit       int
	SortBy      string
	SortAscending bool
}

// PaginatedTransactions represents a paginated list of transactions
type PaginatedTransactions struct {
	Transactions []Transaction `json:"transactions"`
	TotalCount   int           `json:"total_count"`
	Page         int           `json:"page"`
	TotalPages   int           `json:"total_pages"`
}

// NewTransaction creates a new transaction with default values
func NewTransaction(txType TransactionType, sourceID, destID *uuid.UUID, amount float64, description string) *Transaction {
	now := time.Now()
	return &Transaction{
		ID:                 uuid.New(),
		TransactionType:    txType,
		SourceWalletID:     sourceID,
		DestinationWalletID: destID,
		Amount:             amount,
		Status:             Pending,
		Description:        description,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// NewTransactionEvent creates a new transaction event
func NewTransactionEvent(transactionID uuid.UUID, eventType EventType, details map[string]interface{}) *TransactionEvent {
	return &TransactionEvent{
		ID:            uuid.New(),
		TransactionID: transactionID,
		EventType:     eventType,
		Details:       details,
		CreatedAt:     time.Now(),
	}
}