-- +goose Up
-- +goose StatementBegin

CREATE OR REPLACE FUNCTION update_balance_on_transaction()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO balances (wallet_id, currency_id, balance)
    VALUES (NEW.wallet_id, NEW.currency_id, NEW.amount)
    ON CONFLICT (wallet_id, currency_id)
    DO UPDATE SET 
        balance = balances.balance + EXCLUDED.balance,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_balance
AFTER INSERT ON transactions
FOR EACH ROW
EXECUTE FUNCTION update_balance_on_transaction();

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trg_update_balance ON transactions;
DROP FUNCTION IF EXISTS update_balance_on_transaction();

-- +goose StatementEnd
