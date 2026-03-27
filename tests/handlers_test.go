package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/kero-exchange/internal/handlers"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

func TestWalletHandler(t *testing.T) {
	svc := services.NewWalletService(testPool)
	handler := handlers.NewWalletHandler(svc)
	ctx := context.Background()

	t.Run("Get - invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/wallets/invalid", nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("Get - not found", func(t *testing.T) {
		id := uuid.New()
		req := httptest.NewRequest("GET", "/api/v1/wallets/"+id.String(), nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("Get - success", func(t *testing.T) {
		wallet, err := createTestWallet(ctx)
		if err != nil {
			t.Fatalf("Failed to create test wallet: %v", err)
		}
		defer deleteTestWallet(ctx, wallet.UUID)

		req := httptest.NewRequest("GET", "/api/v1/wallets/"+wallet.UUID.String(), nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("List - success", func(t *testing.T) {
		w1, err := createTestWallet(ctx)
		if err != nil {
			t.Fatalf("Failed to create test wallet: %v", err)
		}
		w2, err := createTestWallet(ctx)
		if err != nil {
			t.Fatalf("Failed to create test wallet: %v", err)
		}
		defer deleteTestWallet(ctx, w1.UUID)
		defer deleteTestWallet(ctx, w2.UUID)

		req := httptest.NewRequest("GET", "/api/v1/wallets", nil)
		w := httptest.NewRecorder()
		handler.List(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

func TestBalanceHandler(t *testing.T) {
	svc := services.NewBalanceService(testPool)
	handler := handlers.NewBalanceHandler(svc)
	ctx := context.Background()

	t.Run("Get - invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/balances/invalid", nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("Get - not found", func(t *testing.T) {
		id := uuid.New()
		req := httptest.NewRequest("GET", "/api/v1/balances/"+id.String(), nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("Get - success", func(t *testing.T) {
		wallet, err := createTestWallet(ctx)
		if err != nil {
			t.Fatalf("Failed to create test wallet: %v", err)
		}
		currency, err := createTestCurrency(ctx, "TEST", "Test Currency")
		if err != nil {
			t.Fatalf("Failed to create test currency: %v", err)
		}
		balance, err := createTestBalance(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(100))
		if err != nil {
			t.Fatalf("Failed to create test balance: %v", err)
		}
		defer deleteTestBalance(ctx, balance.UUID)
		defer deleteTestWallet(ctx, wallet.UUID)
		defer deleteTestCurrency(ctx, currency.UUID)

		req := httptest.NewRequest("GET", "/api/v1/balances/"+balance.UUID.String(), nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

func TestTransactionHandler(t *testing.T) {
	svc := services.NewTransactionService(testPool)
	handler := handlers.NewTransactionHandler(svc)
	ctx := context.Background()

	t.Run("Get - invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/transactions/invalid", nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}
	})

	t.Run("Get - not found", func(t *testing.T) {
		id := uuid.New()
		req := httptest.NewRequest("GET", "/api/v1/transactions/"+id.String(), nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("Get - success", func(t *testing.T) {
		wallet, err := createTestWallet(ctx)
		if err != nil {
			t.Fatalf("Failed to create test wallet: %v", err)
		}
		currency, err := createTestCurrency(ctx, "TEST", "Test Currency")
		if err != nil {
			t.Fatalf("Failed to create test currency: %v", err)
		}
		transaction, err := createTestTransaction(ctx, wallet.UUID, currency.UUID, decimal.NewFromInt(100), "DEPOSIT")
		if err != nil {
			t.Fatalf("Failed to create test transaction: %v", err)
		}
		defer deleteTestTransaction(ctx, transaction.UUID)
		defer deleteTestWallet(ctx, wallet.UUID)
		defer deleteTestCurrency(ctx, currency.UUID)

		req := httptest.NewRequest("GET", "/api/v1/transactions/"+transaction.UUID.String(), nil)
		w := httptest.NewRecorder()
		handler.Get(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}
