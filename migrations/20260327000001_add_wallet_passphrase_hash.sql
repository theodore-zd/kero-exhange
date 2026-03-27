-- +goose Up
-- +goose StatementBegin

ALTER TABLE wallet ADD COLUMN IF NOT EXISTS passphrase_hash VARCHAR(64);
CREATE INDEX IF NOT EXISTS idx_wallet_passphrase_hash ON wallet(passphrase_hash);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_wallet_passphrase_hash;
ALTER TABLE wallet DROP COLUMN IF EXISTS passphrase_hash;

-- +goose StatementEnd