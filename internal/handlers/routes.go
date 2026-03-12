package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool) {
	walletSvc := services.NewWalletService(pool)
	currencySvc := services.NewCurrencyService(pool)
	transactionSvc := services.NewTransactionService(pool)

	walletHandler := NewWalletHandler(walletSvc)
	currencyHandler := NewCurrencyHandler(currencySvc)
	transactionHandler := NewTransactionHandler(transactionSvc)

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Get("/health", healthHandler)

	walletHandler.RegisterRoutes(r)
	currencyHandler.RegisterRoutes(r)
	transactionHandler.RegisterRoutes(r)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	common.WriteJSONResponse(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
