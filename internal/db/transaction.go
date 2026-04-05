package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type TransactionType string

const (
	TransactionTypeDeposit     TransactionType = "deposit"
	TransactionTypeWithdrawal  TransactionType = "withdrawal"
	TransactionTypeTransfer    TransactionType = "transfer"
	TransactionTypeAdminIssued TransactionType = "admin_issued"
)

type CreateTransactionParams struct {
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
	Amount     decimal.Decimal
	Type       TransactionType
	Reference  *string
	TransferID *uuid.UUID
}

func CreateTransaction(ctx context.Context, pool *pgxpool.Pool, params CreateTransactionParams) (*Transaction, error) {
	query := `
		INSERT INTO transactions (wallet_id, currency_id, amount, type, reference, transfer_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING uuid, wallet_id, currency_id, amount, type, reference, transfer_id, timestamp
	`
	var t Transaction
	err := pool.QueryRow(ctx, query,
		params.WalletID,
		params.CurrencyID,
		params.Amount,
		params.Type,
		params.Reference,
		params.TransferID,
	).Scan(
		&t.UUID,
		&t.WalletID,
		&t.CurrencyID,
		&t.Amount,
		&t.Type,
		&t.Reference,
		&t.TransferID,
		&t.Timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}
	return &t, nil
}

type Transaction struct {
	UUID       uuid.UUID
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
	Amount     decimal.Decimal
	Type       TransactionType
	Reference  *string
	TransferID *uuid.UUID
	Timestamp  time.Time
	DeletedAt  *time.Time
}

type TransactionFilter struct {
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
	Type       TransactionType
	StartDate  *time.Time
	EndDate    *time.Time
}

func GetTransactionByUUID(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (*Transaction, error) {
	query := `
		SELECT uuid, wallet_id, currency_id, amount, type, reference, transfer_id, timestamp, deleted_at
		FROM transactions
		WHERE uuid = $1 AND deleted_at IS NULL
	`
	var t Transaction
	err := pool.QueryRow(ctx, query, id).Scan(
		&t.UUID,
		&t.WalletID,
		&t.CurrencyID,
		&t.Amount,
		&t.Type,
		&t.Reference,
		&t.TransferID,
		&t.Timestamp,
		&t.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get transaction by uuid: %w", err)
	}
	return &t, nil
}

func GetTransactions(ctx context.Context, pool *pgxpool.Pool, params PaginationParams, filter TransactionFilter) (PaginatedResult[Transaction], error) {
	params = params.Normalize()

	baseQuery := `SELECT uuid, wallet_id, currency_id, amount, type, reference, transfer_id, timestamp, deleted_at FROM transactions WHERE deleted_at IS NULL`
	countQuery := `SELECT COUNT(*) FROM transactions WHERE deleted_at IS NULL`
	args := []any{}
	argIdx := 1
	hasWhere := true

	addCondition := func(condition string, value any) {
		connector := " WHERE "
		if hasWhere {
			connector = " AND "
		}
		where := fmt.Sprintf("%s%s", connector, condition)
		baseQuery += where
		countQuery += where
		args = append(args, value)
		argIdx++
		hasWhere = true
	}

	if filter.WalletID != uuid.Nil {
		addCondition(fmt.Sprintf("wallet_id = $%d", argIdx), filter.WalletID)
	}

	if filter.CurrencyID != uuid.Nil {
		addCondition(fmt.Sprintf("currency_id = $%d", argIdx), filter.CurrencyID)
	}

	if filter.Type != "" {
		addCondition(fmt.Sprintf("type = $%d", argIdx), filter.Type)
	}

	if filter.StartDate != nil {
		addCondition(fmt.Sprintf("timestamp >= $%d", argIdx), filter.StartDate)
	}

	if filter.EndDate != nil {
		addCondition(fmt.Sprintf("timestamp <= $%d", argIdx), filter.EndDate)
	}

	baseQuery += " ORDER BY timestamp DESC"

	return Paginate(ctx, pool, baseQuery, countQuery, args, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]Transaction, error) {
			var transactions []Transaction
			for rows.Next() {
				var t Transaction
				if err := rows.Scan(&t.UUID, &t.WalletID, &t.CurrencyID, &t.Amount, &t.Type, &t.Reference, &t.TransferID, &t.Timestamp, &t.DeletedAt); err != nil {
					return nil, err
				}
				transactions = append(transactions, t)
			}
			return transactions, nil
		},
	)
}

func CreateTransactionTx(ctx context.Context, tx pgx.Tx, params CreateTransactionParams) (*Transaction, error) {
	t := &Transaction{}
	err := tx.QueryRow(ctx,
		`INSERT INTO transactions (wallet_id, currency_id, amount, type, reference, transfer_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING uuid, wallet_id, currency_id, amount, type, reference, transfer_id, timestamp`,
		params.WalletID, params.CurrencyID, params.Amount, params.Type, params.Reference, params.TransferID,
	).Scan(&t.UUID, &t.WalletID, &t.CurrencyID, &t.Amount, &t.Type, &t.Reference, &t.TransferID, &t.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("create transaction tx: %w", err)
	}
	return t, nil
}

func GetTransactionsByTransferID(ctx context.Context, pool *pgxpool.Pool, transferID uuid.UUID) ([]*Transaction, error) {
	rows, err := pool.Query(ctx,
		`SELECT uuid, wallet_id, currency_id, amount, type, reference, transfer_id, timestamp, deleted_at
		 FROM transactions WHERE transfer_id = $1 AND deleted_at IS NULL
		 ORDER BY timestamp DESC`, transferID)
	if err != nil {
		return nil, fmt.Errorf("get transactions by transfer id: %w", err)
	}
	defer rows.Close()

	var txns []*Transaction
	for rows.Next() {
		t := &Transaction{}
		if err := rows.Scan(&t.UUID, &t.WalletID, &t.CurrencyID, &t.Amount, &t.Type,
			&t.Reference, &t.TransferID, &t.Timestamp, &t.DeletedAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, t)
	}
	return txns, nil
}
