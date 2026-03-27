-- +goose Up
-- +goose StatementBegin

ALTER TABLE currencies ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE balances ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE transactions ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_currencies_deleted ON currencies(deleted_at);
CREATE INDEX IF NOT EXISTS idx_balances_deleted ON balances(deleted_at);
CREATE INDEX IF NOT EXISTS idx_transactions_deleted ON transactions(deleted_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_currencies_deleted;
DROP INDEX IF EXISTS idx_balances_deleted;
DROP INDEX IF EXISTS idx_transactions_deleted;

ALTER TABLE currencies DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE balances DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE transactions DROP COLUMN IF EXISTS deleted_at;

-- +goose StatementEnd