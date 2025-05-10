// wallet/model/wallet.go
package model

import (
	"time"
)

type Wallet struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Balance     float64   `json:"balance"`
	IsBlocked   bool      `json:"is_blocked"`
	BlockReason string    `json:"block_reason,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type WalletBalanceUpdate struct {
	Amount float64 `json:"amount"`
}