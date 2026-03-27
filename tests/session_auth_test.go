package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

func TestAuthSignIn_SessionBasedToken(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM access_tokens")
	testPool.Exec(ctx, "DELETE FROM wallet")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	walletUUID := wallet.UUID.String()
	passphrase := walletUUID + "_test_passphrase"

	t.Run("Sign in with correct passphrase", func(t *testing.T) {
		reqBody := strings.Join([]string{
			`{"passphrase":"`,
			passphrase,
			`"}`,
		}, "")
		resp, err := http.Post(server.URL+"/api/v1/auth/signin", "application/json", strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				AccessToken string `json:"access_token"`
				WalletUUID  string `json:"wallet_uuid"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if result.Data.AccessToken == "" {
			t.Error("Expected access token to be returned")
		}

		if result.Data.WalletUUID != walletUUID {
			t.Errorf("Expected wallet UUID %s, got %s", walletUUID, result.Data.WalletUUID)
		}
	})

	t.Run("Verify access token created in database", func(t *testing.T) {
		resp, _ := http.Post(server.URL+"/api/v1/auth/signin", "application/json", strings.NewReader(`{"passphrase":"`+passphrase+`"}`))
		defer resp.Body.Close()

		var result struct {
			Data struct {
				AccessToken string `json:"access_token"`
			} `json:"data"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		token, err := db.GetAccessTokenByToken(ctx, testPool, result.Data.AccessToken)
		if err != nil {
			t.Fatalf("Failed to get access token: %v", err)
		}

		if token == nil {
			t.Error("Expected access token to be stored in database")
		}

		if token.WalletID != wallet.UUID {
			t.Errorf("Expected wallet ID %s, got %s", wallet.UUID, token.WalletID)
		}

		if time.Now().After(token.ExpiresAt) {
			t.Error("Expected token expiration time to be in the future")
		}

		if token.CreatedAt.After(time.Now()) {
			t.Error("Expected token creation time to be in the past or now")
		}
	})
}

func TestAuthSignIn_InvalidPassphrase(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	reqBody := `{"passphrase":"invalid_passphrase"}`
	resp, err := http.Post(server.URL+"/api/v1/auth/signin", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["error"] == nil {
		t.Error("Expected error in response")
	}

	errObj := result["error"].(map[string]interface{})
	if errObj["code"] != "WALLET_NOT_FOUND" {
		t.Errorf("Expected error code WALLET_NOT_FOUND, got %s", errObj["code"])
	}
}

func TestAuthAccessTokenMiddleware_ValidToken(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM access_tokens")
	testPool.Exec(ctx, "DELETE FROM wallet")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	walletUUID := wallet.UUID.String()
	passphrase := walletUUID + "_test_passphrase"

	signInResp, _ := http.Post(server.URL+"/api/v1/auth/signin", "application/json", strings.NewReader(`{"passphrase":"`+passphrase+`"}`))
	defer signInResp.Body.Close()

	var signInResult struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.NewDecoder(signInResp.Body).Decode(&signInResult)

	t.Run("Access protected endpoint with valid token", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/v1/wallets", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+signInResult.Data.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Verify last_used_at updated", func(t *testing.T) {
		time.Sleep(time.Millisecond * 100)

		req, _ := http.NewRequest("GET", server.URL+"/api/v1/wallets", nil)
		req.Header.Set("Authorization", "Bearer "+signInResult.Data.AccessToken)
		resp, _ := http.DefaultClient.Do(req)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
			return
		}

		token, err := db.GetAccessTokenByToken(ctx, testPool, signInResult.Data.AccessToken)
		if err != nil {
			t.Fatalf("Failed to get access token: %v", err)
		}

		if token.LastUsedAt == nil {
			t.Error("Expected last_used_at to be updated")
		}
	})
}

func TestAuthAccessTokenMiddleware_InvalidToken(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	t.Run("Access with invalid token", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/v1/wallets", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer invalid_token_12345")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["error"] == nil {
			t.Error("Expected error in response")
		}

		errObj := result["error"].(map[string]interface{})
		if errObj["code"] != "INVALID_ACCESS_TOKEN" {
			t.Errorf("Expected error code INVALID_ACCESS_TOKEN, got %s", errObj["code"])
		}
	})
}

func TestAuthAccessTokenMiddleware_MissingToken(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	t.Run("Access without authorization header", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/v1/wallets", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["error"] == nil {
			t.Error("Expected error in response")
		}

		errObj := result["error"].(map[string]interface{})
		if errObj["code"] != "MISSING_AUTH_TOKEN" {
			t.Errorf("Expected error code MISSING_AUTH_TOKEN, got %s", errObj["code"])
		}
	})
}

func TestAuthAccessTokenExpired(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM access_tokens")
	testPool.Exec(ctx, "DELETE FROM wallet")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	walletUUID := wallet.UUID.String()
	passphrase := walletUUID + "_test_passphrase"

	server := setupTestServer(t)
	defer server.Close()

	resp, _ := http.Post(server.URL+"/api/v1/auth/signin", "application/json", strings.NewReader(`{"passphrase":"`+passphrase+`"}`))
	if err != nil {
		t.Fatalf("Failed to sign in: %v", err)
	}
	defer resp.Body.Close()

	var signInResult struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&signInResult)

	t.Run("Manually expire token", func(t *testing.T) {
		expiryTime := time.Now().Add(-1 * time.Hour)
		query := `UPDATE access_tokens SET expires_at = $1 WHERE token = $2`
		_, err := testPool.Exec(ctx, query, expiryTime, signInResult.Data.AccessToken)
		if err != nil {
			t.Fatalf("Failed to expire token: %v", err)
		}
	})

	t.Run("Access with expired token", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/api/v1/wallets", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+signInResult.Data.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["error"] == nil {
			t.Error("Expected error in response")
		}

		errObj := result["error"].(map[string]interface{})
		if errObj["code"] != "EXPIRED_ACCESS_TOKEN" {
			t.Errorf("Expected error code EXPIRED_ACCESS_TOKEN, got %s", errObj["code"])
		}
	})
}

func TestAuthSignOut(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM access_tokens")
	testPool.Exec(ctx, "DELETE FROM wallet")

	wallet, err := createTestWallet(ctx)
	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
	defer deleteTestWallet(ctx, wallet.UUID)

	server := setupTestServer(t)
	defer server.Close()

	walletUUID := wallet.UUID.String()
	passphrase := walletUUID + "_test_passphrase"

	resp, _ := http.Post(server.URL+"/api/v1/auth/signin", "application/json", strings.NewReader(`{"passphrase":"`+passphrase+`"}`))
	defer resp.Body.Close()

	var signInResult struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&signInResult)

	t.Run("Sign out deletes token", func(t *testing.T) {
		authSvc := services.NewAuthService(testPool)
		err := authSvc.SignOut(ctx, wallet.UUID)
		if err != nil {
			t.Fatalf("Failed to sign out: %v", err)
		}

		token, err := db.GetAccessTokenByToken(ctx, testPool, signInResult.Data.AccessToken)
		if err != nil {
			t.Fatalf("Failed to check token: %v", err)
		}

		if token != nil {
			t.Error("Expected token to be deleted after sign out")
		}
	})
}
