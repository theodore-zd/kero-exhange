package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

func TestHealthEndpoint(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
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

	if data["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", data["status"])
	}
}

func TestAuthGenerateReferenceCodes_Success(t *testing.T) {
	ctx := context.Background()
	provider, apiKey, err := createTestProvider(ctx, "Test Provider")
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1", provider.UUID)

	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"count": 3, "expires_in_hours": 2}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/providers/reference-codes", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data in response, got: %s", string(body))
	}

	codes, ok := data["codes"].([]interface{})
	if !ok || len(codes) != 3 {
		t.Errorf("Expected 3 codes, got %v", data["codes"])
	}
}

func TestAuthGenerateReferenceCodes_InvalidAPIKey(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"count": 1}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/providers/reference-codes", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "invalid-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAuthGenerateReferenceCodes_MissingAPIKey(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"count": 1}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/providers/reference-codes", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAuthSignup_Success(t *testing.T) {
	ctx := context.Background()
	provider, apiKey, err := createTestProvider(ctx, "Test Provider")
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1", provider.UUID)

	refCode, err := createTestReferenceCode(ctx, provider.UUID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to create reference code: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM reference_codes WHERE uuid = $1", refCode.UUID)

	server := setupTestServer(t)
	defer server.Close()

	publicKey := "0x" + strings.ReplaceAll(uuid.New().String(), "-", "")
	reqBody := `{"public_key": "` + publicKey + `", "reference_code": "` + refCode.Code + `"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signup", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data in response, got: %v", result)
	}

	if data["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
	if data["wallet_uuid"] == nil {
		t.Error("Expected wallet_uuid in response")
	}

	walletUUID, _ := uuid.Parse(data["wallet_uuid"].(string))
	defer deleteTestWallet(ctx, walletUUID)
}

func TestAuthSignup_InvalidReferenceCode(t *testing.T) {
	ctx := context.Background()
	provider, apiKey, err := createTestProvider(ctx, "Test Provider")
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1", provider.UUID)

	server := setupTestServer(t)
	defer server.Close()

	publicKey := "0x" + strings.ReplaceAll(uuid.New().String(), "-", "")
	reqBody := `{"public_key": "` + publicKey + `", "reference_code": "INVALIDCODE"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signup", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAuthSignup_ReferenceCodeUsed(t *testing.T) {
	ctx := context.Background()
	provider, apiKey, err := createTestProvider(ctx, "Test Provider")
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1", provider.UUID)

	refCode, err := createTestReferenceCode(ctx, provider.UUID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to create reference code: %v", err)
	}

	db.MarkReferenceCodeUsed(ctx, testPool, refCode.UUID)
	defer testPool.Exec(ctx, "DELETE FROM reference_codes WHERE uuid = $1", refCode.UUID)

	server := setupTestServer(t)
	defer server.Close()

	publicKey := "0x" + strings.ReplaceAll(uuid.New().String(), "-", "")
	reqBody := `{"public_key": "` + publicKey + `", "reference_code": "` + refCode.Code + `"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signup", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAuthSignup_ReferenceCodeExpired(t *testing.T) {
	ctx := context.Background()
	provider, apiKey, err := createTestProvider(ctx, "Test Provider")
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1", provider.UUID)

	refCode, err := createTestReferenceCode(ctx, provider.UUID, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("Failed to create reference code: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM reference_codes WHERE uuid = $1", refCode.UUID)

	server := setupTestServer(t)
	defer server.Close()

	publicKey := "0x" + strings.ReplaceAll(uuid.New().String(), "-", "")
	reqBody := `{"public_key": "` + publicKey + `", "reference_code": "` + refCode.Code + `"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signup", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestAuthSignup_DuplicatePublicKey(t *testing.T) {
	ctx := context.Background()
	provider, apiKey, err := createTestProvider(ctx, "Test Provider")
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1", provider.UUID)

	refCode, err := createTestReferenceCode(ctx, provider.UUID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("Failed to create reference code: %v", err)
	}
	defer testPool.Exec(ctx, "DELETE FROM reference_codes WHERE uuid = $1", refCode.UUID)

	existingWallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create existing wallet: %v", err)
	}
	defer deleteTestWallet(ctx, existingWallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"public_key": "0x1234567890abcdef", "reference_code": "` + refCode.Code + `"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signup", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp.StatusCode)
	}
}

func TestAuthSignup_MissingAPIKey(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"public_key": "0x123", "reference_code": "TEST"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signup", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestAuthSignin_Success(t *testing.T) {
	ctx := context.Background()
	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"public_key": "0xsigninwallet123"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signin", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data in response, got: %v", result)
	}

	if data["access_token"] == nil {
		t.Error("Expected access_token in response")
	}
	if data["wallet_uuid"] != wallet.UUID.String() {
		t.Errorf("Expected wallet_uuid %s, got %s", wallet.UUID, data["wallet_uuid"])
	}
}

func TestAuthSignin_WalletNotFound(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"public_key": "0xnonexistentwallet123"}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signin", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestAuthSignin_MissingPublicKey(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{}`
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/signin", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}
