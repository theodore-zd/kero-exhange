-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS access_tokens (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallet(uuid) ON DELETE CASCADE,
    token VARCHAR(128) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_access_tokens_token ON access_tokens(token);
CREATE INDEX idx_access_tokens_wallet ON access_tokens(wallet_id);
CREATE INDEX idx_access_tokens_expires ON access_tokens(expires_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_access_tokens_expires;
DROP INDEX IF EXISTS idx_access_tokens_wallet;
DROP INDEX IF EXISTS idx_access_tokens_token;
DROP TABLE IF EXISTS access_tokens;

-- +goose StatementEnd
