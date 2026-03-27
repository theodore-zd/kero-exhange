package tests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/kero-exchange/internal/config"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/handlers"
)

var testPool *pgxpool.Pool

const testDBURL = "postgresql://postgres:postgres@localhost:5443/local_pg?sslmode=disable"

func TestMain(m *testing.M) {
	ctx := context.Background()
	var err error
	testPool, err = pgxpool.New(ctx, testDBURL)
	if err != nil {
		fmt.Printf("Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}
	defer testPool.Close()

	if err := dropAllTables(ctx); err != nil {
		fmt.Printf("Failed to drop tables: %v\n", err)
		os.Exit(1)
	}

	if err := runMigrations(ctx); err != nil {
		fmt.Printf("Failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	cleanupTestData(ctx)
	os.Exit(code)
}

func dropAllTables(ctx context.Context) error {
	tables := []string{
		"DROP TRIGGER IF EXISTS trg_update_balance ON transactions",
		"DROP TABLE IF EXISTS admin_audit_logs",
		"DROP TABLE IF EXISTS access_tokens",
		"DROP TABLE IF EXISTS reference_codes",
		"DROP TABLE IF EXISTS providers",
		"DROP TABLE IF EXISTS transactions",
		"DROP TABLE IF EXISTS balances",
		"DROP TABLE IF EXISTS currencies",
		"DROP TABLE IF EXISTS wallet",
		"DROP FUNCTION IF EXISTS update_balance_on_transaction",
	}
	for _, table := range tables {
		if _, err := testPool.Exec(ctx, table); err != nil {
			return fmt.Errorf("drop table: %w", err)
		}
	}
	return nil
}

func runMigrations(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS wallet (
			uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			access_token_hash VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS currencies (
			uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			code VARCHAR(8) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(500),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS balances (
			uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID NOT NULL,
			currency_id UUID NOT NULL,
			balance DECIMAL(20, 8) NOT NULL DEFAULT 0 CHECK (balance >= 0),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(wallet_id, currency_id),
			FOREIGN KEY (wallet_id) REFERENCES wallet(uuid) ON DELETE CASCADE,
			FOREIGN KEY (currency_id) REFERENCES currencies(uuid) ON DELETE RESTRICT
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID NOT NULL,
			currency_id UUID NOT NULL,
			amount DECIMAL(20, 8) NOT NULL,
			type VARCHAR(20) NOT NULL,
			reference VARCHAR(255),
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			FOREIGN KEY (wallet_id) REFERENCES wallet(uuid) ON DELETE CASCADE,
			FOREIGN KEY (currency_id) REFERENCES currencies(uuid) ON DELETE RESTRICT
		)`,
		`CREATE TABLE IF NOT EXISTS providers (
			uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			api_key_hash VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS reference_codes (
			uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			code VARCHAR(16) NOT NULL UNIQUE,
			provider_id UUID NOT NULL REFERENCES providers(uuid) ON DELETE CASCADE,
			used_at TIMESTAMPTZ,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS access_tokens (
			uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			wallet_id UUID NOT NULL REFERENCES wallet(uuid) ON DELETE CASCADE,
			token VARCHAR(128) NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_used_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS admin_audit_logs (
			uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			action VARCHAR(100) NOT NULL,
			entity_type VARCHAR(50) NOT NULL,
			entity_id UUID,
			details JSONB,
			admin_user VARCHAR(100) DEFAULT 'admin',
			ip_address VARCHAR(45),
			user_agent TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE OR REPLACE FUNCTION update_balance_on_transaction()
		RETURNS TRIGGER AS $$
		BEGIN
			INSERT INTO balances (wallet_id, currency_id, balance)
			VALUES (NEW.wallet_id, NEW.currency_id, NEW.amount)
			ON CONFLICT (wallet_id, currency_id)
			DO UPDATE SET 
				balance = balances.balance + EXCLUDED.balance,
				updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;`,
		`DROP TRIGGER IF EXISTS trg_update_balance ON transactions`,
		`CREATE TRIGGER trg_update_balance
		AFTER INSERT ON transactions
		FOR EACH ROW
		EXECUTE FUNCTION update_balance_on_transaction();`,
	}

	for _, migration := range migrations {
		if _, err := testPool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

func cleanupTestData(ctx context.Context) {
	testPool.Exec(ctx, "DELETE FROM access_tokens")
	testPool.Exec(ctx, "DELETE FROM admin_audit_logs")
	testPool.Exec(ctx, "DELETE FROM reference_codes")
	testPool.Exec(ctx, "DELETE FROM providers")
	testPool.Exec(ctx, "DELETE FROM transactions")
	testPool.Exec(ctx, "DELETE FROM balances")
	testPool.Exec(ctx, "DELETE FROM wallet")
	testPool.Exec(ctx, "DELETE FROM currencies")
}

func setupTestServer(t *testing.T) *httptest.Server {
	r := chi.NewRouter()
	cfg := &config.Config{
		AdminPassword: "test-admin-password",
	}
	handlers.RegisterRoutes(r, testPool, cfg)
	return httptest.NewServer(r)
}

type APIResponse struct {
	Data interface{} `json:"data,omitempty"`
	Meta interface{} `json:"meta,omitempty"`
}

func parseResponse(t *testing.T, body []byte, target interface{}) {
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("Failed to parse response: %v\nResponse: %s", err, string(body))
	}
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func createTestProvider(ctx context.Context, name string) (*db.Provider, string, error) {
	apiKey := uuid.New().String()
	apiKeyHash := hashToken(apiKey)
	provider, err := db.CreateProvider(ctx, testPool, apiKeyHash, name)
	if err != nil {
		return nil, "", err
	}
	return provider, apiKey, nil
}

func createTestCurrency(ctx context.Context, code, name string) (*db.Currency, error) {
	return db.CreateCurrency(ctx, testPool, db.CreateCurrencyParams{
		Code: code,
		Name: name,
	})
}

func createTestWallet(ctx context.Context) (*db.Wallet, error) {
	wallet, err := db.CreateWallet(ctx, testPool, db.CreateWalletParams{
		AccessTokenHash: hashToken(uuid.New().String()),
	})
	if err != nil {
		return nil, err
	}

	walletUUID := wallet.UUID.String()
	passphrase := walletUUID + "_test_passphrase"
	tokenHash := hashToken(walletUUID + passphrase)

	wallet, err = db.CreateWallet(ctx, testPool, db.CreateWalletParams{
		AccessTokenHash: tokenHash,
	})
	return wallet, err
}

func createTestReferenceCode(ctx context.Context, providerID uuid.UUID, expiresAt time.Time) (*db.ReferenceCode, error) {
	code := strings.ToUpper(uuid.New().String()[:16])
	return db.CreateReferenceCode(ctx, testPool, db.CreateReferenceCodeParams{
		Code:       code,
		ProviderID: providerID,
		ExpiresAt:  expiresAt,
	})
}

func deleteTestWallet(ctx context.Context, walletID uuid.UUID) {
	testPool.Exec(ctx, "DELETE FROM wallet WHERE uuid = $1", walletID)
}

func deleteTestCurrency(ctx context.Context, currencyID uuid.UUID) {
	testPool.Exec(ctx, "DELETE FROM currencies WHERE uuid = $1", currencyID)
}

func generateTestAccessToken() string {
	return hex.EncodeToString(make([]byte, 32))
}

func signInTestWallet(server *httptest.Server, passphrase string) (string, string, error) {
	reqBody := fmt.Sprintf(`{"passphrase":"%s"}`, passphrase)
	resp, err := http.Post(server.URL+"/api/v1/auth/signin", "application/json", strings.NewReader(reqBody))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			AccessToken string `json:"access_token"`
			WalletUUID  string `json:"wallet_uuid"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("sign in failed with status %d", resp.StatusCode)
	}

	return result.Data.AccessToken, result.Data.WalletUUID, nil
}

func adminLoginTest(server *httptest.Server, password string) (string, error) {
	reqBody := fmt.Sprintf(`{"password":"%s"}`, password)
	resp, err := http.Post(server.URL+"/api/v1/admin/login", "application/json", strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("admin login failed with status %d", resp.StatusCode)
	}

	return result.Data.Token, nil
}
