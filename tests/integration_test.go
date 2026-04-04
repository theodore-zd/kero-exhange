package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/wispberry-tech/kero-exchange/internal/handlers"
)

func TestCompleteAuthFlow(t *testing.T) {
	ctx := context.Background()
	server := setupTestServer(t)
	defer server.Close()

	provider, apiKey, err := createTestProvider(ctx, "Test Auth Flow Provider")
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1", provider.UUID)

	var referenceCode string
	var walletPassphrase string
	var accessToken string

	t.Run("Generate reference code", func(t *testing.T) {
		req, err := http.NewRequest("POST", server.URL+"/api/v1/providers/reference-codes", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-API-Key", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 201, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				Code string `json:"code"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if result.Data.Code == "" {
			t.Fatal("Expected reference code")
		}
		referenceCode = result.Data.Code
	})

	t.Run("Sign up with reference code", func(t *testing.T) {
		reqBody := `{"reference_code": "` + referenceCode + `"}`
		req, err := http.NewRequest("POST", server.URL+"/api/v1/auth/signup", strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected 201, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				AccessToken      string `json:"access_token"`
				WalletUUID       string `json:"wallet_uuid"`
				SecretPassphrase string `json:"secret_passphrase"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if result.Data.AccessToken == "" {
			t.Error("Expected access_token in signup response")
		}
		if result.Data.WalletUUID == "" {
			t.Error("Expected wallet_uuid in signup response")
		}
		if result.Data.SecretPassphrase == "" {
			t.Fatal("Expected secret_passphrase in signup response")
		}
		walletPassphrase = result.Data.SecretPassphrase
	})

	t.Run("Sign in with passphrase", func(t *testing.T) {
		reqBody := `{"passphrase": "` + walletPassphrase + `"}`
		resp, err := http.Post(server.URL+"/api/v1/auth/signin", "application/json", strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				AccessToken string `json:"access_token"`
				WalletUUID  string `json:"wallet_uuid"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if result.Data.AccessToken == "" {
			t.Fatal("Expected access_token in signin response")
		}
		accessToken = result.Data.AccessToken
	})

	t.Run("Access protected endpoint with token", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/v1/wallets", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestAdminFlow(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	var adminToken string

	t.Run("Admin login", func(t *testing.T) {
		reqBody := `{"password":"test-admin-password"}`
		resp, err := http.Post(server.URL+"/api/v1/admin/login", "application/json", strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("Failed to login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		adminToken = result.Data.Token
	})

	t.Run("Admin login with wrong password", func(t *testing.T) {
		reqBody := `{"password":"wrong-password"}`
		resp, err := http.Post(server.URL+"/api/v1/admin/login", "application/json", strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Create provider", func(t *testing.T) {
		reqBody := `{"name":"Test Provider"}`
		req, err := http.NewRequest("POST", server.URL+"/api/v1/admin/providers", strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 201, got %d", resp.StatusCode)
		}
	})

	t.Run("List providers", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/v1/admin/providers", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Create wallet", func(t *testing.T) {
		req, err := http.NewRequest("POST", server.URL+"/api/v1/admin/wallets", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 201, got %d", resp.StatusCode)
		}

		var result handlers.CreateWalletResponse
		json.NewDecoder(resp.Body).Decode(&result)

		if result.UUID == "" {
			t.Error("Expected wallet UUID")
		}
		if result.Passphrase == "" {
			t.Error("Expected passphrase")
		}
		if result.AccessToken == "" {
			t.Error("Expected access token")
		}
	})
}
