package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDashboardSummary(t *testing.T) {
	ctx := context.Background()

	wallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	currency, err := createTestCurrency(ctx, "DSH", "Dashboard Currency")
	if err != nil {
		t.Fatalf("Failed to create test currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	balance, err := createTestBalance(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(250))
	if err != nil {
		t.Fatalf("Failed to create test balance: %v", err)
	}
	defer deleteTestBalance(ctx, balance.UUID)

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)
	resp := authGet(t, server, token, "/api/v1/dashboard")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			WalletCount int `json:"wallet_count"`
			Balances    []struct {
				CurrencyCode string `json:"currency_code"`
				CurrencyName string `json:"currency_name"`
				TotalAmount  string `json:"total_amount"`
			} `json:"balances"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, string(body))
	}

	if result.Data.WalletCount < 1 {
		t.Errorf("Expected wallet_count >= 1, got %d", result.Data.WalletCount)
	}

	if len(result.Data.Balances) != 1 {
		t.Errorf("Expected 1 balance entry, got %d", len(result.Data.Balances))
	}

	if len(result.Data.Balances) > 0 {
		b := result.Data.Balances[0]
		if b.CurrencyCode != "DSH" {
			t.Errorf("Expected currency_code DSH, got %s", b.CurrencyCode)
		}
		if b.CurrencyName != "Dashboard Currency" {
			t.Errorf("Expected currency_name 'Dashboard Currency', got %s", b.CurrencyName)
		}
		if b.TotalAmount != "250" {
			t.Errorf("Expected total_amount 250, got %s", b.TotalAmount)
		}
	}
}

func TestDashboardSummary_Unauthenticated(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/dashboard")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestDashboardSummary_NoBalances(t *testing.T) {
	ctx := context.Background()

	wallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)
	resp := authGet(t, server, token, "/api/v1/dashboard")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			WalletCount int         `json:"wallet_count"`
			Balances    interface{} `json:"balances"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, string(body))
	}

	if result.Data.WalletCount < 1 {
		t.Errorf("Expected wallet_count >= 1, got %d", result.Data.WalletCount)
	}
}
