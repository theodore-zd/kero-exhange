package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func authPost(t *testing.T, server *httptest.Server, token, path, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	return resp
}

func TestTransferBetweenWallets(t *testing.T) {
	ctx := context.Background()

	sourceWallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create source wallet: %v", err)
	}
	defer deleteTestWallet(ctx, sourceWallet.UUID)

	destWallet, _, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create destination wallet: %v", err)
	}
	defer deleteTestWallet(ctx, destWallet.UUID)

	currency, err := createTestCurrency(ctx, "TRF", "Transfer Currency")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	_, err = createTestBalance(ctx, sourceWallet.UUID, currency.UUID, decimal.NewFromInt(500))
	if err != nil {
		t.Fatalf("Failed to create source balance: %v", err)
	}

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)

	reqBody := fmt.Sprintf(`{"destination_wallet_id":"%s","currency_id":"%s","amount":"100"}`,
		destWallet.UUID.String(), currency.UUID.String())

	resp := authPost(t, server, token, "/api/v1/transfers", reqBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			TransferID string `json:"transfer_id"`
			Debit      struct {
				UUID       string `json:"uuid"`
				WalletID   string `json:"wallet_id"`
				CurrencyID string `json:"currency_id"`
				Amount     string `json:"amount"`
				Type       string `json:"type"`
				Timestamp  string `json:"timestamp"`
			} `json:"debit"`
			Credit struct {
				UUID       string `json:"uuid"`
				WalletID   string `json:"wallet_id"`
				CurrencyID string `json:"currency_id"`
				Amount     string `json:"amount"`
				Type       string `json:"type"`
				Timestamp  string `json:"timestamp"`
			} `json:"credit"`
		} `json:"data"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, string(body))
	}

	if result.Data.TransferID == "" {
		t.Error("Expected transfer_id in response")
	}

	if result.Data.Debit.WalletID != sourceWallet.UUID.String() {
		t.Errorf("Expected debit wallet_id %s, got %s", sourceWallet.UUID, result.Data.Debit.WalletID)
	}
	if result.Data.Credit.WalletID != destWallet.UUID.String() {
		t.Errorf("Expected credit wallet_id %s, got %s", destWallet.UUID, result.Data.Credit.WalletID)
	}
	if result.Data.Debit.Amount != "-100" {
		t.Errorf("Expected debit amount -100, got %s", result.Data.Debit.Amount)
	}
	if result.Data.Credit.Amount != "100" {
		t.Errorf("Expected credit amount 100, got %s", result.Data.Credit.Amount)
	}
	if result.Data.Debit.Type != "transfer" {
		t.Errorf("Expected debit type transfer, got %s", result.Data.Debit.Type)
	}
	if result.Data.Credit.Type != "transfer" {
		t.Errorf("Expected credit type transfer, got %s", result.Data.Credit.Type)
	}
}

func TestTransferInsufficientBalance(t *testing.T) {
	ctx := context.Background()

	sourceWallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create source wallet: %v", err)
	}
	defer deleteTestWallet(ctx, sourceWallet.UUID)

	destWallet, _, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create destination wallet: %v", err)
	}
	defer deleteTestWallet(ctx, destWallet.UUID)

	currency, err := createTestCurrency(ctx, "TRI", "Transfer Insufficient")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	_, err = createTestBalance(ctx, sourceWallet.UUID, currency.UUID, decimal.NewFromInt(50))
	if err != nil {
		t.Fatalf("Failed to create source balance: %v", err)
	}

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)

	reqBody := fmt.Sprintf(`{"destination_wallet_id":"%s","currency_id":"%s","amount":"100"}`,
		destWallet.UUID.String(), currency.UUID.String())

	resp := authPost(t, server, token, "/api/v1/transfers", reqBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	errData, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected error in response, got: %s", string(body))
	}
	if errData["code"] != "INSUFFICIENT_BALANCE" {
		t.Errorf("Expected error code INSUFFICIENT_BALANCE, got %s", errData["code"])
	}
}

func TestTransferToSelf(t *testing.T) {
	ctx := context.Background()

	wallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	currency, err := createTestCurrency(ctx, "TRS", "Transfer Self")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	_, err = createTestBalance(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(500))
	if err != nil {
		t.Fatalf("Failed to create balance: %v", err)
	}

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)

	reqBody := fmt.Sprintf(`{"destination_wallet_id":"%s","currency_id":"%s","amount":"100"}`,
		wallet.UUID.String(), currency.UUID.String())

	resp := authPost(t, server, token, "/api/v1/transfers", reqBody)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 400, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	errData, ok := result["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected error in response, got: %s", string(body))
	}
	if errData["code"] != "SAME_WALLET" {
		t.Errorf("Expected error code SAME_WALLET, got %s", errData["code"])
	}
}

func TestTransferConcurrentDrain(t *testing.T) {
	ctx := context.Background()

	sourceWallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create source wallet: %v", err)
	}
	defer deleteTestWallet(ctx, sourceWallet.UUID)

	destWallet, _, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create destination wallet: %v", err)
	}
	defer deleteTestWallet(ctx, destWallet.UUID)

	currency, err := createTestCurrency(ctx, "TRC", "Transfer Concurrent")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	_, err = createTestBalance(ctx, sourceWallet.UUID, currency.UUID, decimal.NewFromInt(100))
	if err != nil {
		t.Fatalf("Failed to create source balance: %v", err)
	}

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)

	reqBody := fmt.Sprintf(`{"destination_wallet_id":"%s","currency_id":"%s","amount":"100"}`,
		destWallet.UUID.String(), currency.UUID.String())

	// Fire two concurrent transfers that each try to drain the full balance.
	// Exactly one should succeed (201) and the other should fail (400 insufficient balance).
	// Before the fix, the second would hit a DB constraint and return 500.
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func() {
			resp := authPost(t, server, token, "/api/v1/transfers", reqBody)
			defer resp.Body.Close()
			io.ReadAll(resp.Body)
			results <- resp.StatusCode
		}()
	}

	var codes []int
	for i := 0; i < 2; i++ {
		codes = append(codes, <-results)
	}

	successCount := 0
	failCount := 0
	for _, code := range codes {
		switch code {
		case http.StatusCreated:
			successCount++
		case http.StatusBadRequest:
			failCount++
		default:
			t.Errorf("Unexpected status code %d — expected 201 or 400, got 500 indicates a race condition", code)
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful transfer, got %d (codes: %v)", successCount, codes)
	}
	if failCount != 1 {
		t.Errorf("Expected exactly 1 failed transfer, got %d (codes: %v)", failCount, codes)
	}
}

func TestTransfer_Unauthenticated(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"destination_wallet_id":"00000000-0000-0000-0000-000000000001","currency_id":"00000000-0000-0000-0000-000000000002","amount":"10"}`
	resp, err := http.Post(server.URL+"/api/v1/transfers", "application/json", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}
