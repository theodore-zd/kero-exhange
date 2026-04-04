package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestWalletList_Empty(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")

	// Create a wallet just for authentication
	wallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create auth wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)
	resp := authGet(t, server, token, "/api/v1/wallets")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	// At least the auth wallet exists
	if len(data) == 0 {
		t.Error("Expected at least one wallet (the auth wallet)")
	}
}

func TestWalletList_WithData(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")

	wallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)
	resp := authGet(t, server, token, "/api/v1/wallets")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) == 0 {
		t.Error("Expected at least one wallet in data array")
	}
}

func TestWalletList_Pagination(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")

	// Create auth wallet first
	authWallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create auth wallet: %v", err)
	}
	defer deleteTestWallet(ctx, authWallet.UUID)

	for i := 0; i < 24; i++ {
		wallet, _, err := createTestWallet(ctx)
		if err != nil {
			t.Fatalf("Failed to create wallet: %v", err)
		}
		defer deleteTestWallet(ctx, wallet.UUID)
	}

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)
	resp := authGet(t, server, token, "/api/v1/wallets?page=1&page_size=10")
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

func TestWalletGet_Success(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM wallet")

	wallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)
	resp := authGet(t, server, token, "/api/v1/wallets/"+wallet.UUID.String())
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

	if data["uuid"] != wallet.UUID.String() {
		t.Errorf("Expected uuid %s, got %s", wallet.UUID, data["uuid"])
	}
}

func TestWalletGet_NotFound(t *testing.T) {
	ctx := context.Background()
	wallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create auth wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)
	fakeID := uuid.New().String()
	resp := authGet(t, server, token, "/api/v1/wallets/"+fakeID)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestWalletGet_InvalidUUID(t *testing.T) {
	ctx := context.Background()
	wallet, passphrase, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create auth wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	token := getTestAccessToken(t, server, passphrase)
	resp := authGet(t, server, token, "/api/v1/wallets/invalid-uuid")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestWalletList_Unauthenticated(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/wallets")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}
