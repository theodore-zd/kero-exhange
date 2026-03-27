-- +goose Up
-- +goose StatementBegin

CREATE INDEX IF NOT EXISTS idx_wallet_access_token_hash ON wallet(access_token_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_wallet_access_token_hash;

-- +goose StatementEnd