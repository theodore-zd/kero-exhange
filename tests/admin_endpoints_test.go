package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func getAdminToken(t *testing.T, server *httptest.Server) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"password": "test-admin-password"})
	resp, err := http.Post(server.URL+"/api/v1/admin/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to get admin token: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Data.Token == "" {
		t.Fatal("Expected admin token in response")
	}
	return result.Data.Token
}

func adminAuthGet(t *testing.T, server *httptest.Server, token, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", server.URL+path, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	return resp
}

func TestAdminListTransactions(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM transactions")
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")

	wallet, _, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	currency, err := createTestCurrency(ctx, "ADT", "Admin Deposit Test")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(50), "deposit")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	defer deleteTestTransaction(ctx, tx.UUID)

	server := setupTestServer(t)
	defer server.Close()

	adminToken := getAdminToken(t, server)

	resp := adminAuthGet(t, server, adminToken, "/api/v1/admin/transactions")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			Data []interface{}          `json:"data"`
			Meta map[string]interface{} `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, string(body))
	}

	total, ok := result.Data.Meta["total"].(float64)
	if !ok || total < 1 {
		t.Errorf("Expected total >= 1, got %v (body: %s)", result.Data.Meta["total"], string(body))
	}
}

func TestAdminGetTransaction(t *testing.T) {
	ctx := context.Background()

	wallet, _, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	currency, err := createTestCurrency(ctx, "AGT", "Admin Get Tx Test")
	if err != nil {
		t.Fatalf("Failed to create currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	tx, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(25), "deposit")
	if err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}
	defer deleteTestTransaction(ctx, tx.UUID)

	server := setupTestServer(t)
	defer server.Close()

	adminToken := getAdminToken(t, server)

	resp := adminAuthGet(t, server, adminToken, "/api/v1/admin/transactions/"+tx.UUID.String())
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			UUID string `json:"uuid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, string(body))
	}

	if result.Data.UUID != tx.UUID.String() {
		t.Errorf("Expected UUID %s, got %s", tx.UUID.String(), result.Data.UUID)
	}
}

func TestAdminListAuditLogs(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	adminToken := getAdminToken(t, server)

	resp := adminAuthGet(t, server, adminToken, "/api/v1/admin/audit-logs")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			Data []interface{}          `json:"data"`
			Meta map[string]interface{} `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, string(body))
	}

	if _, ok := result.Data.Meta["total"]; !ok {
		t.Error("Expected 'total' field in meta")
	}
	if _, ok := result.Data.Meta["page"]; !ok {
		t.Error("Expected 'page' field in meta")
	}
}

func TestAdminSearchWallets(t *testing.T) {
	ctx := context.Background()

	wallet, _, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	adminToken := getAdminToken(t, server)

	// Search using the first 8 characters of the UUID
	prefix := wallet.UUID.String()[:8]
	resp := adminAuthGet(t, server, adminToken, "/api/v1/admin/wallets/search?q="+prefix)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			Data []map[string]interface{} `json:"data"`
			Meta map[string]interface{}   `json:"meta"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v\nBody: %s", err, string(body))
	}

	found := false
	for _, w := range result.Data.Data {
		if w["uuid"] == wallet.UUID.String() {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected to find wallet %s in search results (prefix: %s), got: %s", wallet.UUID.String(), prefix, string(body))
	}
}

func TestAdminSearchWallets_MissingQuery(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	adminToken := getAdminToken(t, server)

	resp := adminAuthGet(t, server, adminToken, "/api/v1/admin/wallets/search")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
