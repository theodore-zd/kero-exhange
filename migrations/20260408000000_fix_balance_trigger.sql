-- +goose Up
-- +goose StatementBegin

CREATE OR REPLACE FUNCTION update_balance_on_transaction()
RETURNS TRIGGER AS $$
BEGIN
    -- Cannot use INSERT ... ON CONFLICT with negative amounts because
    -- PostgreSQL evaluates CHECK constraints on the proposed INSERT values
    -- BEFORE checking for unique conflicts. A debit (negative amount) would
    -- fail CHECK (balance >= 0) before ON CONFLICT could fire the UPDATE.
    IF EXISTS (SELECT 1 FROM balances WHERE wallet_id = NEW.wallet_id AND currency_id = NEW.currency_id) THEN
        UPDATE balances
        SET balance = balance + NEW.amount,
            updated_at = NOW()
        WHERE wallet_id = NEW.wallet_id AND currency_id = NEW.currency_id;
    ELSE
        INSERT INTO balances (wallet_id, currency_id, balance)
        VALUES (NEW.wallet_id, NEW.currency_id, NEW.amount);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin

-- Restore original ON CONFLICT version
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

-- +goose StatementEnd
