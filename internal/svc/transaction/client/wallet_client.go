package clients

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	walletpb "github.com/yourorg/wallet/wallet/pb/wallet" // Import the wallet proto
)

// WalletServiceClient defines the interface for interacting with the wallet service
type WalletServiceClient interface {
	UpdateBalance(ctx context.Context, walletID uuid.UUID, amount float64, isDebit bool) error
	GetBalance(ctx context.Context, walletID uuid.UUID) (float64, error)
	Close() error
}

// GRPCWalletClient implements WalletServiceClient using gRPC
type GRPCWalletClient struct {
	client walletpb.WalletServiceClient
	conn   *grpc.ClientConn
}

// NewWalletClient creates a new wallet client
func NewWalletClient(walletServiceAddr string) (WalletServiceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		walletServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to wallet service: %w", err)
	}

	client := walletpb.NewWalletServiceClient(conn)
	return &GRPCWalletClient{client: client, conn: conn}, nil
}

// UpdateBalance updates the balance of a wallet
func (c *GRPCWalletClient) UpdateBalance(ctx context.Context, walletID uuid.UUID, amount float64, isDebit bool) error {
	operation := walletpb.BalanceUpdateRequest_CREDIT
	if isDebit {
		operation = walletpb.BalanceUpdateRequest_DEBIT
	}

	req := &walletpb.BalanceUpdateRequest{
		WalletId:  walletID.String(),
		Amount:    amount,
		Operation: operation,
	}

	resp, err := c.client.UpdateBalance(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	if !resp.Success {
		return errors.New(resp.Message)
	}

	return nil
}

// GetBalance retrieves the current balance of a wallet
func (c *GRPCWalletClient) GetBalance(ctx context.Context, walletID uuid.UUID) (float64, error) {
	req := &walletpb.GetWalletRequest{
		WalletId: walletID.String(),
	}

	resp, err := c.client.GetWallet(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to get wallet: %w", err)
	}

	return resp.Wallet.Balance, nil
}

// Close closes the client connection
func (c *GRPCWalletClient) Close() error {
	return c.conn.Close()
}