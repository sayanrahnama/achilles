CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_type VARCHAR(10) NOT NULL CHECK (transaction_type IN ('deposit', 'withdraw', 'transfer')),
    source_wallet_id UUID,
    destination_wallet_id UUID,
    amount DECIMAL(15, 2) NOT NULL CHECK (amount > 0),
    status VARCHAR(10) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    description TEXT,
    saga_state VARCHAR(20) NOT NULL DEFAULT 'STARTED' CHECK (saga_state IN (
        'STARTED', 
        'WALLET_CHECKED',
        'WALLET_UPDATED', 
        'NOTIFICATION_SENT', 
        'COMPLETED', 
        'FAILED', 
        'COMPENSATION_STARTED', 
        'COMPENSATION_COMPLETED'
    )),
    saga_details JSONB,
    idempotency_key VARCHAR(64) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT source_or_destination_check CHECK (
        (transaction_type = 'deposit' AND source_wallet_id IS NULL AND destination_wallet_id IS NOT NULL) OR
        (transaction_type = 'withdraw' AND source_wallet_id IS NOT NULL AND destination_wallet_id IS NULL) OR
        (transaction_type = 'transfer' AND source_wallet_id IS NOT NULL AND destination_wallet_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_transactions_source_wallet_id ON transactions (source_wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_destination_wallet_id ON transactions (destination_wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions (created_at);
CREATE INDEX IF NOT EXISTS idx_transactions_type_status ON transactions (transaction_type, status);
CREATE INDEX IF NOT EXISTS idx_transactions_saga_state ON transactions (saga_state);
CREATE INDEX IF NOT EXISTS idx_transactions_idempotency_key ON transactions (idempotency_key);