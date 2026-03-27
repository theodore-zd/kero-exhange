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
	server := setupTestServer(t)
	defer server.Close()

	t.Run("Generate reference code", func(t *testing.T) {
		ctx := context.Background()
		_, apiKey, err := createTestProvider(ctx, "Test Auth Flow Provider")
		if err != nil {
			t.Fatalf("Failed to create test provider: %v", err)
		}

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
			t.Errorf("Expected 201, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				Code string `json:"code"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if result.Data.Code == "" {
			t.Error("Expected reference code")
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
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		adminToken = result.Data.Token
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
