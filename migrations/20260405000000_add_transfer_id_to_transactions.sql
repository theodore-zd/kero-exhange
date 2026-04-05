-- +goose Up
-- +goose StatementBegin

ALTER TABLE transactions ADD COLUMN transfer_id UUID;
CREATE INDEX idx_transactions_transfer_id ON transactions (transfer_id) WHERE transfer_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_transactions_transfer_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS transfer_id;

-- +goose StatementEnd
