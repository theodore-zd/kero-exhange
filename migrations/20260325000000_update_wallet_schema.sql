-- +goose Up
-- +goose StatementBegin

ALTER TABLE wallet DROP COLUMN IF EXISTS public_key;

DROP INDEX IF EXISTS idx_wallet_public_key;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE wallet ADD COLUMN IF NOT EXISTS public_key VARCHAR(66) NOT NULL UNIQUE;

CREATE INDEX idx_wallet_public_key ON wallet(public_key);

-- +goose StatementEnd
