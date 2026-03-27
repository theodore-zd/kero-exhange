package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

func createTestBalance(ctx context.Context, walletID, currencyID uuid.UUID, balance decimal.Decimal) (*db.Balance, error) {
	query := `
		INSERT INTO balances (wallet_id, currency_id, balance)
		VALUES ($1, $2, $3)
		RETURNING uuid, wallet_id, currency_id, balance, updated_at
	`
	var b db.Balance
	err := testPool.QueryRow(ctx, query, walletID, currencyID, balance).Scan(
		&b.UUID,
		&b.WalletID,
		&b.CurrencyID,
		&b.Balance,
		&b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func deleteTestBalance(ctx context.Context, balanceID uuid.UUID) {
	testPool.Exec(ctx, "DELETE FROM balances WHERE uuid = $1", balanceID)
}

func parseListResponse(t *testing.T, body []byte) (data []interface{}, meta map[string]interface{}) {
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	outerData, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data in response, got: %s", string(body))
	}

	dataObj, ok := outerData["data"].([]interface{})
	if !ok {
		t.Fatalf("Expected data.data in response, got: %s", string(body))
	}

	metaObj, ok := outerData["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data.meta in response, got: %s", string(body))
	}

	return dataObj, metaObj
}

func TestBalanceList_Empty(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM balances")

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/balances")
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

func TestBalanceList_WithData(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "BAL", "Balance Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}

	balance, err := createTestBalance(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(500))
	if err != nil {
		t.Fatalf("Failed to create balance: %v", err)
	}
	defer deleteTestBalance(ctx, balance.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/balances")
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
		t.Error("Expected at least one balance in data array")
	}
}

func TestBalanceList_FilterByWalletID(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "BAL", "Balance Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}

	balance, _ := createTestBalance(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(500))
	defer deleteTestBalance(ctx, balance.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/balances?wallet_id=" + wallet.UUID.String())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) != 1 {
		t.Errorf("Expected 1 balance, got %d", len(data))
	}
}

func TestBalanceList_FilterByCurrencyID(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "BAL", "Balance Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}

	balance, _ := createTestBalance(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(500))
	defer deleteTestBalance(ctx, balance.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/balances?currency_id=" + currency.UUID.String())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) != 1 {
		t.Errorf("Expected 1 balance, got %d", len(data))
	}
}

func TestBalanceList_Pagination(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	for i := 0; i < 25; i++ {
		currency, err := createTestCurrency(ctx, fmt.Sprintf("BLC%02d", i), fmt.Sprintf("Balance Currency %d", i))
		if err != nil {
			t.Fatalf("Failed to create currency: %v", err)
		}
		balance, err := createTestBalance(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(int64(i)*100))
		if err != nil {
			t.Fatalf("Failed to create balance: %v", err)
		}
		defer deleteTestBalance(ctx, balance.UUID)
		defer deleteTestCurrency(ctx, currency.UUID)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/balances?page=1&page_size=10")
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

func TestBalanceGet_Success(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	currency, err := createTestCurrency(ctx, "BAL", "Balance Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	balance, err := createTestBalance(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(500))
	if err != nil {
		t.Fatalf("Failed to create balance: %v", err)
	}
	defer deleteTestBalance(ctx, balance.UUID)
	defer deleteTestWallet(ctx, wallet.UUID)
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/balances/" + balance.UUID.String())
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

	if data["uuid"] != balance.UUID.String() {
		t.Errorf("Expected uuid %s, got %s", balance.UUID, data["uuid"])
	}
	if data["balance"] != "500" {
		t.Errorf("Expected balance 500, got %s", data["balance"])
	}
}

func TestBalanceGet_NotFound(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	fakeID := uuid.New().String()
	resp, err := http.Get(server.URL + "/api/v1/balances/" + fakeID)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestBalanceGet_InvalidUUID(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/balances/invalid-uuid")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
