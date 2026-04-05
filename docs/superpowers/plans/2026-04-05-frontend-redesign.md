# Frontend Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace all Go html/template templates with Grove, redesign UI with terminal aesthetic, add user transfers + admin transaction/audit/lookup views, and dev seed mode.

**Architecture:** Big bang rewrite. New backend endpoints first (TDD), then swap template engine to Grove with new file structure, then build all pages. Dev seed mode as a startup hook gated by env var.

**Tech Stack:** Go, Grove template engine, chi router, pgx/pgxpool, shopspring/decimal, vanilla JS, plain CSS (monospace/terminal theme)

---

### Task 1: Add Grove Dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add grove module**

```bash
go get github.com/wispberry-tech/grove
```

- [ ] **Step 2: Verify import works**

```bash
cd /home/theo/Work/kero-exhange
go build ./...
```

Expected: compiles without errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add grove template engine dependency"
```

---

### Task 2: Database Migration — Transfer ID on Transactions

**Files:**
- Create: `migrations/20260405000000_add_transfer_id_to_transactions.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
ALTER TABLE transactions ADD COLUMN transfer_id UUID;
CREATE INDEX idx_transactions_transfer_id ON transactions (transfer_id) WHERE transfer_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_transactions_transfer_id;
ALTER TABLE transactions DROP COLUMN IF EXISTS transfer_id;
```

- [ ] **Step 2: Run the migration**

```bash
./scripts/migrate.sh up
```

Expected: migration applies successfully.

- [ ] **Step 3: Verify**

```bash
./scripts/migrate.sh status
```

Expected: new migration shows as applied.

- [ ] **Step 4: Commit**

```bash
git add migrations/20260405000000_add_transfer_id_to_transactions.sql
git commit -m "feat: add transfer_id column to transactions table"
```

---

### Task 3: Transfer DB Queries

**Files:**
- Modify: `internal/db/transaction.go`

- [ ] **Step 1: Update Transaction model**

In `internal/db/transaction.go`, add `TransferID` to the model:

```go
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
```

Update `CreateTransactionParams`:

```go
type CreateTransactionParams struct {
	WalletID   uuid.UUID
	CurrencyID uuid.UUID
	Amount     decimal.Decimal
	Type       TransactionType
	Reference  *string
	TransferID *uuid.UUID
}
```

- [ ] **Step 2: Update CreateTransaction to include transfer_id**

Update the INSERT query in `CreateTransaction` to include `transfer_id` in both the column list and values, and scan it back in the RETURNING clause.

- [ ] **Step 3: Update all scan functions to include transfer_id**

Every place that scans a `Transaction` row (`GetTransactionByUUID`, `GetTransactions`, `CreateTransaction`) must now scan `transfer_id` as well. Update the scan call to include `&t.TransferID`.

- [ ] **Step 4: Add CreateTransactionTx function**

```go
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
```

- [ ] **Step 5: Add GetTransactionsByTransferID query**

```go
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
```

- [ ] **Step 6: Run existing tests to verify no regressions**

```bash
go test ./tests/ -v -count=1 2>&1 | tail -20
```

Expected: all existing tests pass (the new nullable column doesn't break existing inserts).

- [ ] **Step 7: Commit**

```bash
git add internal/db/transaction.go
git commit -m "feat: add transfer_id support to transaction model and queries"
```

---

### Task 4: Transfer Service

**Files:**
- Create: `internal/services/transfer.go`
- Create: `tests/transfer_test.go`

- [ ] **Step 1: Write the failing test**

```go
// tests/transfer_test.go
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestTransferBetweenWallets(t *testing.T) {
	ctx := t.Context()
	server := setupTestServer(t)

	walletA, _, tokenA := createTestWallet(ctx)
	walletB, _, _ := createTestWallet(ctx)

	currency := createTestCurrency(ctx, "TXF", "Transfer Test")
	createTestBalance(ctx, walletA.UUID, currency.UUID, "1000.00")

	defer testPool.Exec(ctx, "DELETE FROM transactions")
	defer testPool.Exec(ctx, "DELETE FROM balances")
	defer testPool.Exec(ctx, "DELETE FROM wallets")
	defer testPool.Exec(ctx, "DELETE FROM currencies WHERE code = 'TXF'")

	body, _ := json.Marshal(map[string]interface{}{
		"destination_wallet_id": walletB.UUID.String(),
		"currency_id":          currency.UUID.String(),
		"amount":               "100.00",
	})

	req, _ := http.NewRequest("POST", server.URL+"/api/v1/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var result struct {
		TransferID string `json:"transfer_id"`
		Debit      struct {
			Amount string `json:"amount"`
			Type   string `json:"type"`
		} `json:"debit"`
		Credit struct {
			Amount string `json:"amount"`
			Type   string `json:"type"`
		} `json:"credit"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.TransferID == "" {
		t.Error("expected transfer_id")
	}
	if result.Debit.Type != "transfer" {
		t.Errorf("expected debit type transfer, got %s", result.Debit.Type)
	}
}

func TestTransferInsufficientBalance(t *testing.T) {
	ctx := t.Context()
	server := setupTestServer(t)

	walletA, _, tokenA := createTestWallet(ctx)
	walletB, _, _ := createTestWallet(ctx)
	currency := createTestCurrency(ctx, "TXI", "Insufficient Test")
	createTestBalance(ctx, walletA.UUID, currency.UUID, "50.00")

	defer testPool.Exec(ctx, "DELETE FROM transactions")
	defer testPool.Exec(ctx, "DELETE FROM balances")
	defer testPool.Exec(ctx, "DELETE FROM wallets")
	defer testPool.Exec(ctx, "DELETE FROM currencies WHERE code = 'TXI'")

	body, _ := json.Marshal(map[string]interface{}{
		"destination_wallet_id": walletB.UUID.String(),
		"currency_id":          currency.UUID.String(),
		"amount":               "100.00",
	})

	req, _ := http.NewRequest("POST", server.URL+"/api/v1/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestTransferToSelf(t *testing.T) {
	ctx := t.Context()
	server := setupTestServer(t)

	walletA, _, tokenA := createTestWallet(ctx)
	currency := createTestCurrency(ctx, "TXS", "Self Test")
	createTestBalance(ctx, walletA.UUID, currency.UUID, "1000.00")

	defer testPool.Exec(ctx, "DELETE FROM transactions")
	defer testPool.Exec(ctx, "DELETE FROM balances")
	defer testPool.Exec(ctx, "DELETE FROM wallets")
	defer testPool.Exec(ctx, "DELETE FROM currencies WHERE code = 'TXS'")

	body, _ := json.Marshal(map[string]interface{}{
		"destination_wallet_id": walletA.UUID.String(),
		"currency_id":          currency.UUID.String(),
		"amount":               "100.00",
	})

	req, _ := http.NewRequest("POST", server.URL+"/api/v1/transfers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenA)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test -v ./tests/ -run TestTransfer -count=1 2>&1 | tail -10
```

Expected: compilation errors or 404s (endpoint doesn't exist yet).

- [ ] **Step 3: Write the transfer service**

```go
// internal/services/transfer.go
package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrSameWallet          = errors.New("cannot transfer to the same wallet")
	ErrDestinationNotFound = errors.New("destination wallet not found")
	ErrCurrencyNotFound    = errors.New("currency not found")
)

type TransferService struct {
	pool *pgxpool.Pool
}

func NewTransferService(pool *pgxpool.Pool) *TransferService {
	return &TransferService{pool: pool}
}

type TransferParams struct {
	SourceWalletID uuid.UUID
	DestWalletID   uuid.UUID
	CurrencyID     uuid.UUID
	Amount         decimal.Decimal
}

type TransferResult struct {
	TransferID uuid.UUID
	Debit      *db.Transaction
	Credit     *db.Transaction
}

func (s *TransferService) Transfer(ctx context.Context, params TransferParams) (*TransferResult, error) {
	if params.SourceWalletID == params.DestWalletID {
		return nil, ErrSameWallet
	}
	if params.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}

	destWallet, err := db.GetWalletByUUID(ctx, s.pool, params.DestWalletID)
	if err != nil {
		return nil, fmt.Errorf("lookup destination wallet: %w", err)
	}
	if destWallet == nil {
		return nil, ErrDestinationNotFound
	}

	currency, err := db.GetCurrencyByUUID(ctx, s.pool, params.CurrencyID)
	if err != nil {
		return nil, fmt.Errorf("lookup currency: %w", err)
	}
	if currency == nil {
		return nil, ErrCurrencyNotFound
	}

	balance, err := db.GetBalanceByWalletAndCurrency(ctx, s.pool, params.SourceWalletID, params.CurrencyID)
	if err != nil {
		return nil, fmt.Errorf("lookup source balance: %w", err)
	}
	if balance == nil || balance.Balance.LessThan(params.Amount) {
		return nil, ErrInsufficientBalance
	}

	transferID := uuid.New()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	ref := fmt.Sprintf("transfer:%s", transferID.String())

	debit, err := db.CreateTransactionTx(ctx, tx, db.CreateTransactionParams{
		WalletID:   params.SourceWalletID,
		CurrencyID: params.CurrencyID,
		Amount:     params.Amount.Neg(),
		Type:       db.TransactionTypeTransfer,
		Reference:  &ref,
		TransferID: &transferID,
	})
	if err != nil {
		return nil, fmt.Errorf("create debit transaction: %w", err)
	}

	credit, err := db.CreateTransactionTx(ctx, tx, db.CreateTransactionParams{
		WalletID:   params.DestWalletID,
		CurrencyID: params.CurrencyID,
		Amount:     params.Amount,
		Type:       db.TransactionTypeTransfer,
		Reference:  &ref,
		TransferID: &transferID,
	})
	if err != nil {
		return nil, fmt.Errorf("create credit transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transfer: %w", err)
	}

	return &TransferResult{
		TransferID: transferID,
		Debit:      debit,
		Credit:     credit,
	}, nil
}
```

- [ ] **Step 4: Write the transfer handler**

```go
// internal/handlers/transfer.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	common "github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/middleware/context"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type TransferHandler struct {
	svc *services.TransferService
}

func NewTransferHandler(svc *services.TransferService) *TransferHandler {
	return &TransferHandler{svc: svc}
}

type TransferRequest struct {
	DestinationWalletID string `json:"destination_wallet_id"`
	CurrencyID          string `json:"currency_id"`
	Amount              string `json:"amount"`
}

type TransferTransactionResponse struct {
	UUID       string `json:"uuid"`
	WalletID   string `json:"wallet_id"`
	CurrencyID string `json:"currency_id"`
	Amount     string `json:"amount"`
	Type       string `json:"type"`
	Timestamp  string `json:"timestamp"`
}

type TransferResponse struct {
	TransferID string                      `json:"transfer_id"`
	Debit      TransferTransactionResponse `json:"debit"`
	Credit     TransferTransactionResponse `json:"credit"`
}

func (h *TransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
		return
	}

	destWalletID, err := uuid.Parse(req.DestinationWalletID)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid destination wallet ID", nil)
		return
	}

	currencyID, err := uuid.Parse(req.CurrencyID)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid currency ID", nil)
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid amount", nil)
		return
	}

	sourceWalletID := context.GetWalletUUID(r.Context())

	result, err := h.svc.Transfer(r.Context(), services.TransferParams{
		SourceWalletID: sourceWalletID,
		DestWalletID:   destWalletID,
		CurrencyID:     currencyID,
		Amount:         amount,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInsufficientBalance):
			common.WriteJSONError(w, http.StatusBadRequest, "insufficient_balance", "Insufficient balance", nil)
		case errors.Is(err, services.ErrSameWallet):
			common.WriteJSONError(w, http.StatusBadRequest, "same_wallet", "Cannot transfer to the same wallet", nil)
		case errors.Is(err, services.ErrDestinationNotFound):
			common.WriteJSONError(w, http.StatusBadRequest, "destination_not_found", "Destination wallet not found", nil)
		case errors.Is(err, services.ErrCurrencyNotFound):
			common.WriteJSONError(w, http.StatusBadRequest, "currency_not_found", "Currency not found", nil)
		default:
			handleServiceError(w, err)
		}
		return
	}

	resp := TransferResponse{
		TransferID: result.TransferID.String(),
		Debit: TransferTransactionResponse{
			UUID:       result.Debit.UUID.String(),
			WalletID:   result.Debit.WalletID.String(),
			CurrencyID: result.Debit.CurrencyID.String(),
			Amount:     result.Debit.Amount.String(),
			Type:       string(result.Debit.Type),
			Timestamp:  result.Debit.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		},
		Credit: TransferTransactionResponse{
			UUID:       result.Credit.UUID.String(),
			WalletID:   result.Credit.WalletID.String(),
			CurrencyID: result.Credit.CurrencyID.String(),
			Amount:     result.Credit.Amount.String(),
			Type:       string(result.Credit.Type),
			Timestamp:  result.Credit.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		},
	}

	common.WriteJSONResponse(w, http.StatusCreated, resp)
}
```

- [ ] **Step 5: Register the transfer route**

In `internal/handlers/routes.go`, add to the access-token-protected group:

```go
transferSvc := services.NewTransferService(pool)
transferHandler := NewTransferHandler(transferSvc)
// Inside the r.Group with AccessTokenMiddleware:
r.Post("/api/v1/transfers", transferHandler.Create)
```

- [ ] **Step 6: Run the transfer tests**

```bash
go test -v ./tests/ -run TestTransfer -count=1
```

Expected: all three tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/services/transfer.go internal/handlers/transfer.go internal/handlers/routes.go tests/transfer_test.go
git commit -m "feat: add user-to-user transfer endpoint with validation"
```

---

### Task 5: Dashboard Summary Endpoint

**Files:**
- Create: `internal/services/dashboard.go`
- Create: `internal/handlers/dashboard.go`
- Create: `tests/dashboard_test.go`
- Modify: `internal/db/balance.go`
- Modify: `internal/handlers/routes.go`

- [ ] **Step 1: Write the failing test**

```go
// tests/dashboard_test.go
package tests

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestDashboardSummary(t *testing.T) {
	ctx := t.Context()
	server := setupTestServer(t)

	wallet, _, token := createTestWallet(ctx)
	currency := createTestCurrency(ctx, "DSH", "Dashboard Test")
	createTestBalance(ctx, wallet.UUID, currency.UUID, "500.00")

	defer testPool.Exec(ctx, "DELETE FROM transactions")
	defer testPool.Exec(ctx, "DELETE FROM balances")
	defer testPool.Exec(ctx, "DELETE FROM wallets")
	defer testPool.Exec(ctx, "DELETE FROM currencies WHERE code = 'DSH'")

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		WalletCount int `json:"wallet_count"`
		Balances    []struct {
			CurrencyCode string `json:"currency_code"`
			TotalAmount  string `json:"total_amount"`
		} `json:"balances"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.WalletCount != 1 {
		t.Errorf("expected 1 wallet, got %d", result.WalletCount)
	}
	if len(result.Balances) != 1 {
		t.Fatalf("expected 1 balance, got %d", len(result.Balances))
	}
	if result.Balances[0].CurrencyCode != "DSH" {
		t.Errorf("expected DSH, got %s", result.Balances[0].CurrencyCode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -v ./tests/ -run TestDashboard -count=1 2>&1 | tail -5
```

Expected: 404 or compilation error.

- [ ] **Step 3: Add DB query for balance summary**

In `internal/db/balance.go`:

```go
type BalanceSummary struct {
	CurrencyCode string
	CurrencyName string
	TotalAmount  decimal.Decimal
}

func GetBalanceSummaryByWallet(ctx context.Context, pool *pgxpool.Pool, walletID uuid.UUID) ([]BalanceSummary, error) {
	rows, err := pool.Query(ctx,
		`SELECT c.code, c.name, COALESCE(SUM(b.balance), 0) as total
		 FROM balances b
		 JOIN currencies c ON c.uuid = b.currency_id
		 WHERE b.wallet_id = $1 AND b.deleted_at IS NULL AND c.deleted_at IS NULL
		 GROUP BY c.code, c.name
		 ORDER BY c.code`, walletID)
	if err != nil {
		return nil, fmt.Errorf("get balance summary: %w", err)
	}
	defer rows.Close()

	var summaries []BalanceSummary
	for rows.Next() {
		var s BalanceSummary
		if err := rows.Scan(&s.CurrencyCode, &s.CurrencyName, &s.TotalAmount); err != nil {
			return nil, fmt.Errorf("scan balance summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}
```

- [ ] **Step 4: Write dashboard service**

```go
// internal/services/dashboard.go
package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

type DashboardService struct {
	pool *pgxpool.Pool
}

func NewDashboardService(pool *pgxpool.Pool) *DashboardService {
	return &DashboardService{pool: pool}
}

type DashboardSummary struct {
	WalletCount int
	Balances    []db.BalanceSummary
}

func (s *DashboardService) GetSummary(ctx context.Context, walletID uuid.UUID) (*DashboardSummary, error) {
	balances, err := db.GetBalanceSummaryByWallet(ctx, s.pool, walletID)
	if err != nil {
		return nil, fmt.Errorf("get balance summary: %w", err)
	}

	return &DashboardSummary{
		WalletCount: 1,
		Balances:    balances,
	}, nil
}
```

- [ ] **Step 5: Write dashboard handler**

```go
// internal/handlers/dashboard.go
package handlers

import (
	"net/http"

	common "github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/middleware/context"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type DashboardHandler struct {
	svc *services.DashboardService
}

func NewDashboardHandler(svc *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

type BalanceSummaryResponse struct {
	CurrencyCode string `json:"currency_code"`
	CurrencyName string `json:"currency_name"`
	TotalAmount  string `json:"total_amount"`
}

type DashboardResponse struct {
	WalletCount int                      `json:"wallet_count"`
	Balances    []BalanceSummaryResponse  `json:"balances"`
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	walletID := context.GetWalletUUID(r.Context())

	summary, err := h.svc.GetSummary(r.Context(), walletID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	balances := make([]BalanceSummaryResponse, len(summary.Balances))
	for i, b := range summary.Balances {
		balances[i] = BalanceSummaryResponse{
			CurrencyCode: b.CurrencyCode,
			CurrencyName: b.CurrencyName,
			TotalAmount:  b.TotalAmount.String(),
		}
	}

	common.WriteJSONResponse(w, http.StatusOK, DashboardResponse{
		WalletCount: summary.WalletCount,
		Balances:    balances,
	})
}
```

- [ ] **Step 6: Register the route**

In `internal/handlers/routes.go`, in the access-token-protected group:

```go
dashboardSvc := services.NewDashboardService(pool)
dashboardHandler := NewDashboardHandler(dashboardSvc)
r.Get("/api/v1/dashboard", dashboardHandler.Summary)
```

- [ ] **Step 7: Run tests**

```bash
go test -v ./tests/ -run TestDashboard -count=1
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/db/balance.go internal/services/dashboard.go internal/handlers/dashboard.go internal/handlers/routes.go tests/dashboard_test.go
git commit -m "feat: add dashboard summary endpoint"
```

---

### Task 6: Admin Transaction List Endpoint

**Files:**
- Modify: `internal/handlers/admin_api.go`
- Modify: `internal/handlers/routes.go`
- Create: `tests/admin_transaction_test.go`

- [ ] **Step 1: Write the failing test**

```go
// tests/admin_transaction_test.go
package tests

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAdminListTransactions(t *testing.T) {
	ctx := t.Context()
	server := setupTestServer(t)
	adminToken := getAdminToken(t, server)

	wallet, _, _ := createTestWallet(ctx)
	currency := createTestCurrency(ctx, "ATX", "Admin Tx Test")
	createTestTransaction(ctx, wallet.UUID, currency.UUID, "100.00", "deposit")

	defer testPool.Exec(ctx, "DELETE FROM transactions")
	defer testPool.Exec(ctx, "DELETE FROM balances")
	defer testPool.Exec(ctx, "DELETE FROM wallets")
	defer testPool.Exec(ctx, "DELETE FROM currencies WHERE code = 'ATX'")

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Data []json.RawMessage `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Meta.Total < 1 {
		t.Errorf("expected at least 1 transaction, got %d", result.Meta.Total)
	}
}
```

Note: If `getAdminToken` helper doesn't exist, add it to `tests/api_test.go`:

```go
func getAdminToken(t *testing.T, server *httptest.Server) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"password": "admin"})
	resp, err := http.Post(server.URL+"/api/v1/admin/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("admin login failed: %v", err)
	}
	defer resp.Body.Close()
	var result struct{ Token string `json:"token"` }
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Token == "" {
		t.Fatal("failed to get admin token")
	}
	return result.Token
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -v ./tests/ -run TestAdminListTransactions -count=1 2>&1 | tail -5
```

Expected: 404 or method not allowed.

- [ ] **Step 3: Add ListTransactions and GetTransaction to AdminAPIHandler**

In `internal/handlers/admin_api.go`, the handler needs access to `pool`. Add `pool *pgxpool.Pool` to the `AdminAPIHandler` struct if not already there, and update the constructor.

```go
func (h *AdminAPIHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     parseQueryInt(r, "page", 1),
		PageSize: parseQueryInt(r, "page_size", 20),
	}
	params.Normalize()

	filter := db.TransactionFilter{}
	if wid := r.URL.Query().Get("wallet_id"); wid != "" {
		if id, err := uuid.Parse(wid); err == nil {
			filter.WalletID = id
		}
	}
	if cid := r.URL.Query().Get("currency_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil {
			filter.CurrencyID = id
		}
	}
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = db.TransactionType(t)
	}
	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.StartDate = &t
		}
	}
	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.EndDate = &t
		}
	}

	svc := services.NewTransactionService(h.pool)
	result, err := svc.GetAll(r.Context(), params, filter)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	data := make([]TransactionResponse, len(result.Data))
	for i, t := range result.Data {
		data[i] = toTransactionResponse(t)
	}

	common.WriteJSONResponse(w, http.StatusOK, TransactionListResponse{
		Data: data,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func (h *AdminAPIHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDOrError(w, r, "id", "invalid_id", "Invalid transaction ID")
	if err != nil {
		return
	}

	svc := services.NewTransactionService(h.pool)
	txn, err := svc.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if txn == nil {
		handleNotFoundError(w, "transaction")
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, toTransactionResponse(txn))
}
```

- [ ] **Step 4: Add `parseQueryInt` helper if not present**

In `internal/handlers/helpers.go`:

```go
func parseQueryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
```

- [ ] **Step 5: Register admin transaction routes**

In `internal/handlers/routes.go`, inside `registerAdminProtectedRoutes`:

```go
r.Get("/api/v1/admin/transactions", adminAPIHandler.ListTransactions)
r.Get("/api/v1/admin/transactions/{id}", adminAPIHandler.GetTransaction)
```

- [ ] **Step 6: Run tests**

```bash
go test -v ./tests/ -run TestAdminListTransactions -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/handlers/admin_api.go internal/handlers/helpers.go internal/handlers/routes.go tests/admin_transaction_test.go
git commit -m "feat: add admin transaction list and detail endpoints"
```

---

### Task 7: Admin Audit Log Endpoint

**Files:**
- Modify: `internal/handlers/admin_api.go`
- Modify: `internal/handlers/routes.go`
- Create: `tests/admin_audit_log_test.go`

- [ ] **Step 1: Write the failing test**

```go
// tests/admin_audit_log_test.go
package tests

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAdminListAuditLogs(t *testing.T) {
	ctx := t.Context()
	server := setupTestServer(t)
	adminToken := getAdminToken(t, server)

	defer testPool.Exec(ctx, "DELETE FROM admin_audit_logs")

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/audit-logs", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Action    string `json:"action"`
			AdminUser string `json:"admin_user"`
		} `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Meta.Total < 0 {
		t.Error("total should not be negative")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -v ./tests/ -run TestAdminListAuditLogs -count=1 2>&1 | tail -5
```

Expected: 404.

- [ ] **Step 3: Add ListAuditLogs to AdminAPIHandler**

In `internal/handlers/admin_api.go`:

```go
type AuditLogResponse struct {
	UUID       string                 `json:"uuid"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   *string                `json:"entity_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	AdminUser  string                 `json:"admin_user"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	CreatedAt  string                 `json:"created_at"`
}

type AuditLogListResponse struct {
	Data []AuditLogResponse `json:"data"`
	Meta PaginationMeta     `json:"meta"`
}

func (h *AdminAPIHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     parseQueryInt(r, "page", 1),
		PageSize: parseQueryInt(r, "page_size", 20),
	}
	params.Normalize()

	filter := db.AuditLogFilter{}
	if a := r.URL.Query().Get("action"); a != "" {
		filter.Action = a
	}
	if et := r.URL.Query().Get("entity_type"); et != "" {
		filter.EntityType = et
	}

	auditSvc := services.NewAuditLogService(h.pool)
	result, err := auditSvc.GetAuditLogs(r.Context(), params, filter)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	data := make([]AuditLogResponse, len(result.Data))
	for i, a := range result.Data {
		resp := AuditLogResponse{
			UUID:       a.UUID.String(),
			Action:     a.Action,
			EntityType: a.EntityType,
			Details:    a.Details,
			AdminUser:  a.AdminUser,
			IPAddress:  a.IPAddress,
			UserAgent:  a.UserAgent,
			CreatedAt:  a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if a.EntityID != nil {
			eid := a.EntityID.String()
			resp.EntityID = &eid
		}
		data[i] = resp
	}

	common.WriteJSONResponse(w, http.StatusOK, AuditLogListResponse{
		Data: data,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}
```

- [ ] **Step 4: Register the route**

In `internal/handlers/routes.go`, inside `registerAdminProtectedRoutes`:

```go
r.Get("/api/v1/admin/audit-logs", adminAPIHandler.ListAuditLogs)
```

- [ ] **Step 5: Run tests**

```bash
go test -v ./tests/ -run TestAdminListAuditLogs -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/handlers/admin_api.go internal/handlers/routes.go tests/admin_audit_log_test.go
git commit -m "feat: add admin audit log list endpoint"
```

---

### Task 8: Admin Wallet Search Endpoint

**Files:**
- Modify: `internal/db/wallet.go`
- Modify: `internal/handlers/admin_api.go`
- Modify: `internal/handlers/routes.go`
- Create: `tests/admin_wallet_search_test.go`

- [ ] **Step 1: Write the failing test**

```go
// tests/admin_wallet_search_test.go
package tests

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAdminSearchWallets(t *testing.T) {
	ctx := t.Context()
	server := setupTestServer(t)
	adminToken := getAdminToken(t, server)

	wallet, _, _ := createTestWallet(ctx)

	defer testPool.Exec(ctx, "DELETE FROM wallets")

	prefix := wallet.UUID.String()[:8]
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/wallets/search?q="+prefix, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			UUID string `json:"uuid"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Data) < 1 {
		t.Error("expected at least 1 result")
	}

	found := false
	for _, w := range result.Data {
		if w.UUID == wallet.UUID.String() {
			found = true
		}
	}
	if !found {
		t.Error("expected to find the test wallet in search results")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -v ./tests/ -run TestAdminSearchWallets -count=1 2>&1 | tail -5
```

Expected: 404.

- [ ] **Step 3: Add SearchWallets DB query**

In `internal/db/wallet.go`:

```go
func SearchWallets(ctx context.Context, pool *pgxpool.Pool, query string, params PaginationParams) (*PaginatedResult[*Wallet], error) {
	params.Normalize()
	likeQuery := query + "%"

	baseQuery := `SELECT uuid, passphrase_hash, access_token_hash, created_at, updated_at
		FROM wallets WHERE uuid::text LIKE $1 ORDER BY created_at DESC`
	countQuery := `SELECT COUNT(*) FROM wallets WHERE uuid::text LIKE $1`

	return Paginate[*Wallet](ctx, pool, baseQuery, countQuery,
		[]interface{}{likeQuery}, params.PageSize, params.Offset(),
		func(rows pgx.Rows) (*Wallet, error) {
			w := &Wallet{}
			err := rows.Scan(&w.UUID, &w.PassphraseHash, &w.AccessTokenHash, &w.CreatedAt, &w.UpdatedAt)
			return w, err
		})
}
```

- [ ] **Step 4: Add SearchWallets handler method**

In `internal/handlers/admin_api.go`:

```go
func (h *AdminAPIHandler) SearchWallets(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "missing_query", "Query parameter 'q' is required", nil)
		return
	}

	params := db.PaginationParams{
		Page:     parseQueryInt(r, "page", 1),
		PageSize: parseQueryInt(r, "page_size", 20),
	}
	params.Normalize()

	result, err := db.SearchWallets(r.Context(), h.pool, q, params)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	data := make([]WalletResponse, len(result.Data))
	for i, wallet := range result.Data {
		data[i] = toWalletResponse(wallet)
	}

	common.WriteJSONResponse(w, http.StatusOK, WalletListResponse{
		Data: data,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}
```

- [ ] **Step 5: Register the route**

In `internal/handlers/routes.go`, inside `registerAdminProtectedRoutes`. This route must be registered BEFORE `/api/v1/admin/wallets/{id}` to avoid chi treating "search" as an `{id}` parameter:

```go
r.Get("/api/v1/admin/wallets/search", adminAPIHandler.SearchWallets)
```

- [ ] **Step 6: Run tests**

```bash
go test -v ./tests/ -run TestAdminSearchWallets -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/db/wallet.go internal/handlers/admin_api.go internal/handlers/routes.go tests/admin_wallet_search_test.go
git commit -m "feat: add admin wallet search endpoint"
```

---

### Task 9: Dev Seed Mode

**Files:**
- Modify: `internal/config/config.go`
- Create: `internal/services/seed.go`
- Modify: `cmd/server/main.go`
- Create: `tests/seed_test.go`

- [ ] **Step 1: Write the failing test**

```go
// tests/seed_test.go
package tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

func TestSeedMode(t *testing.T) {
	ctx := context.Background()

	seedSvc := services.NewSeedService(testPool)
	err := seedSvc.Seed(ctx)
	if err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	// Verify seed wallet A exists
	walletA, err := db.GetWalletByUUID(ctx, testPool, uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	if err != nil {
		t.Fatalf("get wallet A: %v", err)
	}
	if walletA == nil {
		t.Fatal("seed wallet A not found")
	}

	// Verify seed wallet B exists
	walletB, err := db.GetWalletByUUID(ctx, testPool, uuid.MustParse("00000000-0000-0000-0000-000000000002"))
	if err != nil {
		t.Fatalf("get wallet B: %v", err)
	}
	if walletB == nil {
		t.Fatal("seed wallet B not found")
	}

	// Verify idempotency
	err = seedSvc.Seed(ctx)
	if err != nil {
		t.Fatalf("second seed should be idempotent: %v", err)
	}

	// Cleanup
	testPool.Exec(ctx, "DELETE FROM transactions")
	testPool.Exec(ctx, "DELETE FROM balances")
	testPool.Exec(ctx, "DELETE FROM access_tokens")
	testPool.Exec(ctx, "DELETE FROM wallets WHERE uuid IN ($1, $2)",
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"))
	testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1",
		uuid.MustParse("00000000-0000-0000-0000-000000000003"))
	testPool.Exec(ctx, "DELETE FROM currencies WHERE code IN ('USD', 'EUR')")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -v ./tests/ -run TestSeedMode -count=1 2>&1 | tail -5
```

Expected: compilation error (SeedService doesn't exist).

- [ ] **Step 3: Add SeedMode to config**

In `internal/config/config.go`, add `SeedMode bool` to the `Config` struct and in `Load()`:

```go
cfg.SeedMode = os.Getenv("SEED_MODE") == "true"
```

- [ ] **Step 4: Write the seed service**

```go
// internal/services/seed.go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	common "github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/crypto"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

var (
	seedWalletAUUID  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	seedWalletBUUID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	seedProviderUUID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

type SeedService struct {
	pool *pgxpool.Pool
}

func NewSeedService(pool *pgxpool.Pool) *SeedService {
	return &SeedService{pool: pool}
}

func (s *SeedService) Seed(ctx context.Context) error {
	existing, err := db.GetWalletByUUID(ctx, s.pool, seedWalletAUUID)
	if err != nil {
		return fmt.Errorf("check seed wallet: %w", err)
	}
	if existing != nil {
		common.Logger.Info("Seed data already exists, skipping")
		return nil
	}

	common.Logger.Info("[SEED] Dev seed mode enabled — creating test data")

	// Create currencies
	usd, err := s.ensureCurrency(ctx, "USD", "US Dollar", "United States Dollar")
	if err != nil {
		return fmt.Errorf("seed USD: %w", err)
	}
	eur, err := s.ensureCurrency(ctx, "EUR", "Euro", "European Euro")
	if err != nil {
		return fmt.Errorf("seed EUR: %w", err)
	}

	// Create wallet A
	passphraseA := "test-user-alpha"
	passphraseHashA := crypto.HashPassphrase(passphraseA)
	tokenA := "kero_test-token-alpha-000000000000000000000000000000000000"
	tokenHashA := crypto.HashAccessToken(seedWalletAUUID, tokenA)

	_, err = s.pool.Exec(ctx,
		`INSERT INTO wallets (uuid, passphrase_hash, access_token_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())`,
		seedWalletAUUID, passphraseHashA, tokenHashA)
	if err != nil {
		return fmt.Errorf("create seed wallet A: %w", err)
	}

	// Create wallet B
	passphraseB := "test-user-beta"
	passphraseHashB := crypto.HashPassphrase(passphraseB)
	tokenB := "kero_test-token-beta-0000000000000000000000000000000000000"
	tokenHashB := crypto.HashAccessToken(seedWalletBUUID, tokenB)

	_, err = s.pool.Exec(ctx,
		`INSERT INTO wallets (uuid, passphrase_hash, access_token_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())`,
		seedWalletBUUID, passphraseHashB, tokenHashB)
	if err != nil {
		return fmt.Errorf("create seed wallet B: %w", err)
	}

	// Create access tokens (far future expiry for dev convenience)
	farFuture := time.Now().Add(365 * 24 * time.Hour)
	if err := s.createAccessToken(ctx, seedWalletAUUID, tokenHashA, farFuture); err != nil {
		return fmt.Errorf("create seed token A: %w", err)
	}
	if err := s.createAccessToken(ctx, seedWalletBUUID, tokenHashB, farFuture); err != nil {
		return fmt.Errorf("create seed token B: %w", err)
	}

	// Fund wallets via transactions (balance trigger handles balance rows)
	if err := s.fundWallet(ctx, seedWalletAUUID, usd.UUID, "10000.00"); err != nil {
		return err
	}
	if err := s.fundWallet(ctx, seedWalletAUUID, eur.UUID, "5000.00"); err != nil {
		return err
	}
	if err := s.fundWallet(ctx, seedWalletBUUID, usd.UUID, "5000.00"); err != nil {
		return err
	}
	if err := s.fundWallet(ctx, seedWalletBUUID, eur.UUID, "2500.00"); err != nil {
		return err
	}

	// Create test provider
	providerAPIKey := "kero_test-provider-key-00000000000000000000000000000000"
	providerKeyHash := crypto.HashPassphrase(providerAPIKey)
	_, err = s.pool.Exec(ctx,
		`INSERT INTO providers (uuid, api_key_hash, name, created_at) VALUES ($1, $2, $3, NOW())`,
		seedProviderUUID, providerKeyHash, "Test Provider")
	if err != nil {
		return fmt.Errorf("create seed provider: %w", err)
	}

	common.Logger.Info(fmt.Sprintf("[SEED] Test Wallet A: passphrase=%s uuid=%s", passphraseA, seedWalletAUUID))
	common.Logger.Info(fmt.Sprintf("[SEED] Test Wallet B: passphrase=%s uuid=%s", passphraseB, seedWalletBUUID))
	common.Logger.Info(fmt.Sprintf("[SEED] Test Provider:  api-key=%s", providerAPIKey))

	return nil
}

func (s *SeedService) ensureCurrency(ctx context.Context, code, name, description string) (*db.Currency, error) {
	existing, err := db.GetCurrencyByCode(ctx, s.pool, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}
	desc := description
	return db.CreateCurrency(ctx, s.pool, db.CreateCurrencyParams{
		Code: code, Name: name, Description: &desc,
	})
}

func (s *SeedService) createAccessToken(ctx context.Context, walletID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := db.CreateAccessToken(ctx, s.pool, db.CreateAccessTokenParams{
		WalletID:  walletID,
		Token:     tokenHash,
		ExpiresAt: expiresAt,
	})
	return err
}

func (s *SeedService) fundWallet(ctx context.Context, walletID, currencyID uuid.UUID, amount string) error {
	amt, _ := decimal.NewFromString(amount)
	ref := "seed-funding"
	_, err := db.CreateTransaction(ctx, s.pool, db.CreateTransactionParams{
		WalletID:   walletID,
		CurrencyID: currencyID,
		Amount:     amt,
		Type:       db.TransactionTypeDeposit,
		Reference:  &ref,
	})
	return err
}
```

- [ ] **Step 5: Integrate seed into server startup**

In `cmd/server/main.go`, after the default currency setup and before route registration:

```go
if cfg.SeedMode {
	seedSvc := services.NewSeedService(pool)
	if err := seedSvc.Seed(ctx); err != nil {
		common.Logger.Error(fmt.Sprintf("Failed to seed: %v", err))
	}
}
```

- [ ] **Step 6: Run the seed test**

```bash
go test -v ./tests/ -run TestSeedMode -count=1
```

Expected: PASS.

- [ ] **Step 7: Run all tests to verify no regressions**

```bash
go test ./tests/ -count=1 2>&1 | tail -5
```

Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add internal/config/config.go internal/services/seed.go cmd/server/main.go tests/seed_test.go
git commit -m "feat: add dev seed mode with test wallets, currencies, and provider"
```

---

### Task 10: Grove Rendering Helper

**Files:**
- Create: `internal/handlers/render.go`
- Modify: `internal/handlers/routes.go`

- [ ] **Step 1: Create the Grove rendering helper**

```go
// internal/handlers/render.go
package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wispberry-tech/grove"
	common "github.com/wispberry-tech/go-common"
)

var groveEngine *grove.Engine

func InitGrove() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	templatesDir := filepath.Join(dir, "templates")
	store := grove.NewFileSystemStore(templatesDir)
	groveEngine = grove.New(
		grove.WithStore(store),
		grove.WithCacheSize(256),
	)

	return nil
}

func renderGrove(w http.ResponseWriter, r *http.Request, template string, data grove.Data) {
	if data == nil {
		data = grove.Data{}
	}

	result, err := groveEngine.Render(r.Context(), template, data)
	if err != nil {
		common.Logger.Error(fmt.Sprintf("template render error: %v", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(result.Body))
}
```

- [ ] **Step 2: Initialize Grove in route registration**

In `internal/handlers/routes.go`, at the start of `RegisterRoutes`:

```go
if err := InitGrove(); err != nil {
	common.Logger.Error(fmt.Sprintf("Failed to initialize grove: %v", err))
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./cmd/server/
```

Expected: compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/render.go internal/handlers/routes.go
git commit -m "feat: add Grove template rendering helper"
```

---

### Task 11: Base Template, CSS, and JS

**Files:**
- Create: `templates/base.grov`
- Rewrite: `static/css/main.css`
- Rewrite: `static/js/app.js`
- Rewrite: `static/js/admin.js`

- [ ] **Step 1: Write base.grov**

```jinja2
{# templates/base.grov #}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{% block title %}kero-exchange{% endblock %}</title>
    <link rel="stylesheet" href="/static/css/main.css">
    {% block head %}{% endblock %}
</head>
<body>
    {% block nav %}{% endblock %}
    <main>
        {% block content %}{% endblock %}
    </main>
    {% block scripts %}{% endblock %}
</body>
</html>
```

- [ ] **Step 2: Write terminal-aesthetic CSS**

```css
/* static/css/main.css */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

:root {
    --bg: #1a1a1a;
    --bg-alt: #222222;
    --fg: #e0e0e0;
    --fg-dim: #888888;
    --accent: #00ff88;
    --link: #00cccc;
    --error: #ff4444;
    --border: #333333;
    --font: "SF Mono", "Cascadia Code", "Fira Code", Consolas, monospace;
}

html { font-size: 14px; }
body {
    font-family: var(--font);
    background: var(--bg);
    color: var(--fg);
    line-height: 1.6;
    min-height: 100vh;
}

a { color: var(--link); text-decoration: none; }
a:hover { text-decoration: underline; }

.top-nav {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 0.75rem 1.5rem;
    border-bottom: 1px solid var(--border);
    background: var(--bg-alt);
}
.top-nav .brand { color: var(--accent); font-weight: bold; }
.top-nav a { color: var(--fg-dim); }
.top-nav a:hover, .top-nav a.active { color: var(--accent); text-decoration: none; }
.top-nav .spacer { flex: 1; }

.layout-sidebar { display: flex; min-height: calc(100vh - 42px); }
.sidebar {
    width: 200px;
    padding: 1rem;
    border-right: 1px solid var(--border);
    background: var(--bg-alt);
}
.sidebar a { display: block; padding: 0.25rem 0; color: var(--fg-dim); }
.sidebar a:hover, .sidebar a.active { color: var(--accent); text-decoration: none; }
.sidebar-content { flex: 1; padding: 1.5rem; overflow-x: auto; }

main { padding: 1.5rem; max-width: 960px; }

h1 { font-size: 1.4rem; color: var(--accent); margin-bottom: 1rem; }
h2 { font-size: 1.1rem; color: var(--fg); margin-bottom: 0.75rem; }
p { margin-bottom: 0.75rem; }
.dim { color: var(--fg-dim); }

table { width: 100%; border-collapse: collapse; margin-bottom: 1rem; }
th, td { text-align: left; padding: 0.4rem 0.75rem; border-bottom: 1px solid var(--border); }
th { color: var(--fg-dim); font-weight: normal; text-transform: uppercase; font-size: 0.85rem; }
tr:hover td { background: var(--bg-alt); }

label { display: block; color: var(--fg-dim); margin-bottom: 0.25rem; font-size: 0.85rem; }
input, select, textarea {
    width: 100%;
    padding: 0.5rem;
    background: var(--bg-alt);
    border: 1px solid var(--border);
    color: var(--fg);
    font-family: var(--font);
    font-size: 0.9rem;
    margin-bottom: 0.75rem;
}
input:focus, select:focus, textarea:focus { outline: none; border-color: var(--accent); }

button, .btn {
    display: inline-block;
    padding: 0.5rem 1rem;
    background: transparent;
    border: 1px solid var(--accent);
    color: var(--accent);
    font-family: var(--font);
    font-size: 0.9rem;
    cursor: pointer;
}
button:hover, .btn:hover { background: var(--accent); color: var(--bg); }
.btn-danger { border-color: var(--error); color: var(--error); }
.btn-danger:hover { background: var(--error); color: var(--bg); }

.alert { padding: 0.75rem; margin-bottom: 1rem; border: 1px solid var(--border); }
.alert-error { border-color: var(--error); color: var(--error); }
.alert-success { border-color: var(--accent); color: var(--accent); }

.pagination { display: flex; gap: 0.5rem; margin-top: 1rem; }
.pagination a, .pagination span { padding: 0.25rem 0.5rem; border: 1px solid var(--border); color: var(--fg-dim); }
.pagination a:hover { border-color: var(--accent); color: var(--accent); text-decoration: none; }
.pagination .active { border-color: var(--accent); color: var(--accent); }

.mb-1 { margin-bottom: 0.5rem; }
.mb-2 { margin-bottom: 1rem; }
.mt-1 { margin-top: 0.5rem; }
.mt-2 { margin-top: 1rem; }
.text-right { text-align: right; }
.text-center { text-align: center; }
.flex { display: flex; }
.flex-between { display: flex; justify-content: space-between; align-items: center; }
.gap-1 { gap: 0.5rem; }

.stats { display: flex; gap: 1.5rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
.stat { padding: 1rem; border: 1px solid var(--border); min-width: 150px; }
.stat-value { font-size: 1.4rem; color: var(--accent); }
.stat-label { font-size: 0.85rem; color: var(--fg-dim); }

@media (max-width: 600px) {
    .layout-sidebar { flex-direction: column; }
    .sidebar { width: 100%; border-right: none; border-bottom: 1px solid var(--border); }
    .stats { flex-direction: column; }
}
```

- [ ] **Step 3: Write app.js**

```javascript
// static/js/app.js
function getAccessToken() { return localStorage.getItem('accessToken'); }
function setAccessToken(token) { localStorage.setItem('accessToken', token); }
function getWalletUUID() { return localStorage.getItem('walletUUID'); }
function setWalletUUID(uuid) { localStorage.setItem('walletUUID', uuid); }

function signOut() {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('walletUUID');
    window.location.href = '/signin';
}

async function apiFetch(url, opts = {}) {
    const token = getAccessToken();
    if (!token) { signOut(); return; }

    const headers = { 'Authorization': 'Bearer ' + token, ...opts.headers };
    if (opts.body && typeof opts.body === 'object') {
        headers['Content-Type'] = 'application/json';
        opts.body = JSON.stringify(opts.body);
    }

    const resp = await fetch(url, { ...opts, headers });
    if (resp.status === 401) { signOut(); return; }
    return resp;
}
```

- [ ] **Step 4: Write admin.js**

```javascript
// static/js/admin.js
function getAdminToken() { return localStorage.getItem('adminToken'); }
function setAdminToken(token) { localStorage.setItem('adminToken', token); }

function adminSignOut() {
    localStorage.removeItem('adminToken');
    window.location.href = '/admin/login';
}

async function adminFetch(url, opts = {}) {
    const token = getAdminToken();
    if (!token) { adminSignOut(); return; }

    const headers = { 'Authorization': 'Bearer ' + token, ...opts.headers };
    if (opts.body && typeof opts.body === 'object') {
        headers['Content-Type'] = 'application/json';
        opts.body = JSON.stringify(opts.body);
    }

    const resp = await fetch(url, { ...opts, headers });
    if (resp.status === 401) { adminSignOut(); return; }
    return resp;
}
```

- [ ] **Step 5: Commit**

```bash
git add templates/base.grov static/css/main.css static/js/app.js static/js/admin.js
git commit -m "feat: add Grove base template, terminal-aesthetic CSS, and auth JS helpers"
```

---

### Task 12: Grove Components

**Files:**
- Create: `templates/components/nav.grov`
- Create: `templates/components/pagination.grov`
- Create: `templates/components/alert.grov`

- [ ] **Step 1: User navigation component**

```jinja2
{# templates/components/nav.grov #}
<nav class="top-nav">
    <span class="brand">kero-exchange</span>
    <a href="/dashboard" class="{% if active == 'dashboard' %}active{% endif %}">dashboard</a>
    <a href="/wallets" class="{% if active == 'wallets' %}active{% endif %}">wallets</a>
    <a href="/transactions" class="{% if active == 'transactions' %}active{% endif %}">transactions</a>
    <a href="/transfer" class="{% if active == 'transfer' %}active{% endif %}">transfer</a>
    <span class="spacer"></span>
    <a href="#" onclick="signOut(); return false;">sign out</a>
</nav>
```

- [ ] **Step 2: Pagination component**

```jinja2
{# templates/components/pagination.grov #}
{% if total_pages > 1 %}
<div class="pagination">
    {% if page > 1 %}
        <a href="?page={{ page - 1 }}&page_size={{ page_size }}">&lt; prev</a>
    {% endif %}
    {% for p in 1..total_pages + 1 %}
        {% if p == page %}
            <span class="active">{{ p }}</span>
        {% else %}
            <a href="?page={{ p }}&page_size={{ page_size }}">{{ p }}</a>
        {% endif %}
    {% endfor %}
    {% if page < total_pages %}
        <a href="?page={{ page + 1 }}&page_size={{ page_size }}">next &gt;</a>
    {% endif %}
</div>
{% endif %}
```

- [ ] **Step 3: Alert component**

```jinja2
{# templates/components/alert.grov #}
{% if error %}
<div class="alert alert-error">{{ error }}</div>
{% endif %}
{% if success %}
<div class="alert alert-success">{{ success }}</div>
{% endif %}
```

- [ ] **Step 4: Commit**

```bash
git add templates/components/
git commit -m "feat: add Grove reusable components (nav, pagination, alert)"
```

---

### Task 13: User Pages

**Files:**
- Create: `templates/user/signin.grov`
- Create: `templates/user/dashboard.grov`
- Create: `templates/user/wallets.grov`
- Create: `templates/user/wallet-detail.grov`
- Create: `templates/user/transactions.grov`
- Create: `templates/user/transfer.grov`
- Modify: `internal/handlers/web.go`

- [ ] **Step 1: Write all user templates**

Each template extends `base.grov`, includes `components/nav.grov` in the `nav` block, and uses client-side JS to fetch data from the API. See the design spec for page content. Templates follow this pattern:

```jinja2
{% extends "base.grov" %}
{% block title %}page name — kero-exchange{% endblock %}
{% block nav %}{% include "components/nav.grov" active="page_name" %}{% endblock %}
{% block content %}
    <!-- page-specific HTML with loading states and data containers -->
{% endblock %}
{% block scripts %}
<script src="/static/js/app.js"></script>
<script>
    // Fetch data from API, populate DOM
</script>
{% endblock %}
```

**signin.grov** — passphrase form, POST to `/api/v1/auth/signin`, store token + redirect to `/dashboard`

**dashboard.grov** — fetch GET `/api/v1/dashboard`, show stats (wallet count, currencies) and balance table

**wallets.grov** — fetch GET `/api/v1/wallets?page=N`, show paginated table with links to detail

**wallet-detail.grov** — fetch wallet, balances, transactions by wallet ID from URL path. Show wallet info, balance table, transaction table, link to transfer

**transactions.grov** — fetch GET `/api/v1/transactions?page=N&type=X`, show filterable paginated table

**transfer.grov** — load currencies dropdown from GET `/api/v1/currencies`. Form with destination wallet UUID, currency select, amount. `confirm()` dialog before POST to `/api/v1/transfers`. Show result on success.

- [ ] **Step 2: Rewrite WebHandler to use Grove**

```go
// internal/handlers/web.go
package handlers

import (
	"net/http"

	"github.com/wispberry-tech/grove"
)

type WebHandler struct{}

func NewWebHandler() *WebHandler {
	return &WebHandler{}
}

func (h *WebHandler) SignInPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/signin.grov", nil)
}

func (h *WebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/dashboard.grov", nil)
}

func (h *WebHandler) WalletsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/wallets.grov", nil)
}

func (h *WebHandler) WalletDetailPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/wallet-detail.grov", nil)
}

func (h *WebHandler) TransactionsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/transactions.grov", nil)
}

func (h *WebHandler) TransferPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/transfer.grov", nil)
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add templates/user/ internal/handlers/web.go
git commit -m "feat: add all user-facing Grove pages"
```

---

### Task 14: Admin Pages

**Files:**
- Create: `templates/admin/base.grov`
- Create: `templates/admin/login.grov`
- Create: `templates/admin/dashboard.grov`
- Create: `templates/admin/providers.grov`
- Create: `templates/admin/provider-form.grov`
- Create: `templates/admin/wallets.grov`
- Create: `templates/admin/wallet-detail.grov`
- Create: `templates/admin/wallet-form.grov`
- Create: `templates/admin/currencies.grov`
- Create: `templates/admin/currency-form.grov`
- Create: `templates/admin/transactions.grov`
- Create: `templates/admin/audit-log.grov`
- Create: `templates/admin/user-lookup.grov`
- Modify: `internal/handlers/admin_web.go`

- [ ] **Step 1: Write admin base template**

```jinja2
{# templates/admin/base.grov #}
{% extends "base.grov" %}

{% block content %}
<div class="layout-sidebar">
    <aside class="sidebar">
        <div class="brand mb-2">kero admin</div>
        <a href="/admin/dashboard" class="{% if active == 'dashboard' %}active{% endif %}">dashboard</a>
        <a href="/admin/providers" class="{% if active == 'providers' %}active{% endif %}">providers</a>
        <a href="/admin/wallets" class="{% if active == 'wallets' %}active{% endif %}">wallets</a>
        <a href="/admin/currencies" class="{% if active == 'currencies' %}active{% endif %}">currencies</a>
        <a href="/admin/transactions" class="{% if active == 'transactions' %}active{% endif %}">transactions</a>
        <a href="/admin/audit-log" class="{% if active == 'audit-log' %}active{% endif %}">audit log</a>
        <a href="/admin/lookup" class="{% if active == 'lookup' %}active{% endif %}">lookup</a>
        <a href="#" onclick="adminSignOut(); return false;">logout</a>
    </aside>
    <div class="sidebar-content">
        {% block admin_content %}{% endblock %}
    </div>
</div>
{% endblock %}
```

- [ ] **Step 2: Write all admin templates**

Each admin page (except login) extends `admin/base.grov`, sets `active` variable, and uses `admin_content` block. Login extends `base.grov` directly (no sidebar). All use `adminFetch()` from `admin.js` for API calls.

**login.grov** — password form, POST to `/api/v1/admin/login`, store token, redirect

**dashboard.grov** — fetch provider/wallet/currency counts via `?page_size=1` (using meta.total), show stats

**providers.grov** — paginated list, create/edit/delete links. Delete with `confirm()`.

**provider-form.grov** — accepts `mode` variable ("create" or "edit") and `provider_id`. Create: name input + POST. Edit: PUT to regenerate API key. Shows key once on success.

**wallets.grov** — paginated list with detail/regen/issue/delete actions

**wallet-detail.grov** — accepts `wallet_id` variable. Shows balances and transactions for that wallet.

**wallet-form.grov** — accepts `mode` ("create", "regenerate", "issue") and `wallet_id`. Create: POST, shows passphrase/token. Regenerate: POST, shows new passphrase. Issue: currency dropdown + amount, POST.

**currencies.grov** — paginated list with delete action

**currency-form.grov** — code/name/description inputs, POST to create

**transactions.grov** — paginated filterable list (type dropdown, wallet UUID input)

**audit-log.grov** — paginated filterable list (action type dropdown)

**user-lookup.grov** — search input, search button, results table with links to wallet detail

- [ ] **Step 3: Rewrite AdminWebHandler to use Grove**

```go
// internal/handlers/admin_web.go
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wispberry-tech/grove"
)

type AdminWebHandler struct{}

func NewAdminWebHandler() *AdminWebHandler {
	return &AdminWebHandler{}
}

func (h *AdminWebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/login.grov", nil)
}

func (h *AdminWebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/dashboard.grov", nil)
}

func (h *AdminWebHandler) ProvidersPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/providers.grov", nil)
}

func (h *AdminWebHandler) ProviderCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/provider-form.grov", grove.Data{"mode": "create"})
}

func (h *AdminWebHandler) ProviderEditPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/provider-form.grov", grove.Data{"mode": "edit", "provider_id": id})
}

func (h *AdminWebHandler) WalletsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/wallets.grov", nil)
}

func (h *AdminWebHandler) WalletCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "create"})
}

func (h *AdminWebHandler) WalletDetailPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-detail.grov", grove.Data{"wallet_id": id})
}

func (h *AdminWebHandler) WalletRegeneratePage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "regenerate", "wallet_id": id})
}

func (h *AdminWebHandler) WalletIssueCurrencyPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "issue", "wallet_id": id})
}

func (h *AdminWebHandler) CurrenciesPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/currencies.grov", nil)
}

func (h *AdminWebHandler) CurrencyCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/currency-form.grov", nil)
}

func (h *AdminWebHandler) TransactionsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/transactions.grov", nil)
}

func (h *AdminWebHandler) AuditLogPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/audit-log.grov", nil)
}

func (h *AdminWebHandler) LookupPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/user-lookup.grov", nil)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./cmd/server/
```

- [ ] **Step 5: Commit**

```bash
git add templates/admin/ internal/handlers/admin_web.go
git commit -m "feat: add all admin Grove pages with sidebar layout"
```

---

### Task 15: Update Route Registration

**Files:**
- Modify: `internal/handlers/routes.go`

- [ ] **Step 1: Update all web routes**

Add new user web routes:
```go
r.Get("/", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/signin", http.StatusFound) })
r.Get("/signin", webHandler.SignInPage)
r.Get("/dashboard", webHandler.DashboardPage)
r.Get("/wallets", webHandler.WalletsPage)
r.Get("/wallets/{id}", webHandler.WalletDetailPage)
r.Get("/transactions", webHandler.TransactionsPage)
r.Get("/transfer", webHandler.TransferPage)
```

Add new admin web routes:
```go
r.Get("/admin", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/admin/dashboard", http.StatusFound) })
r.Get("/admin/login", adminWebHandler.LoginPage)
r.Get("/admin/dashboard", adminWebHandler.DashboardPage)
r.Get("/admin/providers", adminWebHandler.ProvidersPage)
r.Get("/admin/providers/new", adminWebHandler.ProviderCreatePage)
r.Get("/admin/providers/{id}/edit-api-key", adminWebHandler.ProviderEditPage)
r.Get("/admin/wallets", adminWebHandler.WalletsPage)
r.Get("/admin/wallets/new", adminWebHandler.WalletCreatePage)
r.Get("/admin/wallets/{id}", adminWebHandler.WalletDetailPage)
r.Get("/admin/wallets/{id}/regenerate", adminWebHandler.WalletRegeneratePage)
r.Get("/admin/wallets/{id}/issue-currency", adminWebHandler.WalletIssueCurrencyPage)
r.Get("/admin/currencies", adminWebHandler.CurrenciesPage)
r.Get("/admin/currencies/new", adminWebHandler.CurrencyCreatePage)
r.Get("/admin/transactions", adminWebHandler.TransactionsPage)
r.Get("/admin/audit-log", adminWebHandler.AuditLogPage)
r.Get("/admin/lookup", adminWebHandler.LookupPage)
```

Update constructor calls (no more service dependency for web handlers):
```go
webHandler := NewWebHandler()
adminWebHandler := NewAdminWebHandler()
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./cmd/server/
```

- [ ] **Step 3: Run all tests**

```bash
go test ./tests/ -count=1 2>&1 | tail -10
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/handlers/routes.go
git commit -m "feat: update route registration for all new pages and endpoints"
```

---

### Task 16: Remove Old Templates and Cleanup

**Files:**
- Delete: all `.html` files in `templates/` and `templates/admin/`
- Delete: `static/js/auth.js`

- [ ] **Step 1: Remove old HTML templates**

```bash
rm -f templates/base.html templates/signin.html templates/wallets.html
rm -f templates/admin/login.html templates/admin/dashboard.html templates/admin/providers.html
rm -f templates/admin/provider_create.html templates/admin/provider_edit.html
rm -f templates/admin/wallets.html templates/admin/wallet_create.html
rm -f templates/admin/wallet_regenerate.html templates/admin/wallet_issue_currency.html
rm -f templates/admin/currencies.html templates/admin/currency_create.html
```

- [ ] **Step 2: Remove old auth.js**

```bash
rm -f static/js/auth.js
```

- [ ] **Step 3: Verify build and tests**

```bash
go build ./cmd/server/ && go test ./tests/ -count=1 2>&1 | tail -10
```

Expected: compiles and all tests pass.

- [ ] **Step 4: Run vet and format**

```bash
go fmt ./... && go vet ./...
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: remove old html/template files, replaced by Grove templates"
```

---

### Task 17: Final Integration Verification

- [ ] **Step 1: Run full test suite**

```bash
go test -v ./tests/ -count=1
```

Expected: all tests pass.

- [ ] **Step 2: Run race detection**

```bash
go test -race ./tests/ -count=1
```

Expected: no races detected.

- [ ] **Step 3: Manual smoke test**

```bash
SEED_MODE=true ./scripts/dev.sh
```

Verify in browser:
- `/signin` loads with terminal aesthetic
- Sign in with `test-user-alpha`
- `/dashboard` shows stats and balances
- `/wallets` lists wallets
- `/wallets/{id}` shows detail with balances and transactions
- `/transactions` shows filterable list
- `/transfer` form works end-to-end (transfer from alpha to beta)
- `/admin/login` loads, login works
- All admin pages load with sidebar
- Admin transactions, audit log, and lookup pages work
