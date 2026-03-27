package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

func createTestTransaction(ctx context.Context, walletID, currencyID uuid.UUID, amount decimal.Decimal, txType string) (*db.Transaction, error) {
	return db.CreateTransaction(ctx, testPool, db.CreateTransactionParams{
		WalletID:   walletID,
		CurrencyID: currencyID,
		Amount:     amount,
		Type:       db.TransactionType(txType),
	})
}

func deleteTestTransaction(ctx context.Context, txID uuid.UUID) {
	testPool.Exec(ctx, "DELETE FROM transactions WHERE uuid = $1", txID)
}

func TestTransactionList_Empty(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM transactions")

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/transactions")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) != 0 {
		t.Errorf("Expected empty data array, got %d items", len(data))
	}
}

func TestTransactionList_WithData(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "TXS", "Test Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(100), "deposit")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	defer deleteTestTransaction(ctx, tx.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/transactions")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) == 0 {
		t.Error("Expected at least one transaction in data array")
	}
}

func TestTransactionList_FilterByWalletID(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "TXS", "Test Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(100), "deposit")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	defer deleteTestTransaction(ctx, tx.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/transactions?wallet_id=" + wallet.UUID.String())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(data))
	}
}

func TestTransactionList_FilterByCurrencyID(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "TXS", "Test Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(100), "deposit")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	defer deleteTestTransaction(ctx, tx.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/transactions?currency_id=" + currency.UUID.String())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(data))
	}
}

func TestTransactionList_FilterByType(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "TXS", "Test Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(100), "deposit")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	defer deleteTestTransaction(ctx, tx.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/transactions?type=deposit")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(data))
	}
}

func TestTransactionList_FilterByDateRange(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "TXS", "Test Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(100), "deposit")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	defer deleteTestTransaction(ctx, tx.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	startDate := time.Now().Add(-time.Hour).Format("2006-01-02T15:04:05Z")
	endDate := time.Now().Add(time.Hour).Format("2006-01-02T15:04:05Z")

	resp, err := http.Get(server.URL + "/api/v1/transactions?start_date=" + startDate + "&end_date=" + endDate)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(data))
	}
}

func TestTransactionList_Pagination(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "TXS", "Test Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}

	for i := 0; i < 25; i++ {
		tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(int64(i)), "deposit")
		if err != nil {
			t.Fatalf("Failed to create transaction: %v", err)
		}
		defer deleteTestTransaction(ctx, tx.UUID)
	}
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/transactions?page=1&page_size=10")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	_, meta := parseListResponse(t, body)

	if meta["page"] != float64(1) {
		t.Errorf("Expected page 1, got %v", meta["page"])
	}
	if meta["page_size"] != float64(10) {
		t.Errorf("Expected page_size 10, got %v", meta["page_size"])
	}
}

func TestTransactionGet_Success(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "TXS", "Test Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(100), "deposit")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	defer deleteTestTransaction(ctx, tx.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/transactions/" + tx.UUID.String())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data in response, got: %s", string(body))
	}

	if data["uuid"] != tx.UUID.String() {
		t.Errorf("Expected uuid %s, got %s", tx.UUID, data["uuid"])
	}
}

func TestTransactionGet_NotFound(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	fakeID := uuid.New().String()
	resp, err := http.Get(server.URL + "/api/v1/transactions/" + fakeID)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestTransactionGet_InvalidUUID(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/transactions/invalid-uuid")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
