CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    balance DECIMAL(19, 4) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    is_blocked BOOLEAN NOT NULL DEFAULT false,
    block_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_user_wallet UNIQUE (user_id)
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);