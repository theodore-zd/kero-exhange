-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS wallet (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_key VARCHAR(66) NOT NULL UNIQUE,
    access_token_hash VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallet_public_key ON wallet(public_key);

CREATE TABLE IF NOT EXISTS currencies (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(8) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS balances (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL,
    currency_id UUID NOT NULL,
    balance DECIMAL(20, 8) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(wallet_id, currency_id),
    FOREIGN KEY (wallet_id) REFERENCES wallet(uuid) ON DELETE CASCADE,
    FOREIGN KEY (currency_id) REFERENCES currencies(uuid) ON DELETE RESTRICT
);

CREATE INDEX idx_balances_wallet ON balances(wallet_id);
CREATE INDEX idx_balances_wallet_currency ON balances(wallet_id, currency_id);

CREATE TABLE IF NOT EXISTS transactions (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL,
    currency_id UUID NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    type VARCHAR(20) NOT NULL,
    reference VARCHAR(255),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (wallet_id) REFERENCES wallet(uuid) ON DELETE CASCADE,
    FOREIGN KEY (currency_id) REFERENCES currencies(uuid) ON DELETE RESTRICT
);

CREATE INDEX idx_transactions_wallet ON transactions(wallet_id, timestamp DESC);
CREATE INDEX idx_transactions_currency ON transactions(currency_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS balances;
DROP TABLE IF EXISTS currencies;
DROP TABLE IF EXISTS wallet;

-- +goose StatementEnd
