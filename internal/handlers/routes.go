package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/config"
	authMiddleware "github.com/wispberry-tech/kero-exchange/internal/middleware"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

const (
	RouteSignUp           = "/api/v1/auth/signup"
	RouteSignIn           = "/api/v1/auth/signin"
	RouteRefCodeGen       = "/api/v1/providers/reference-codes"
	RouteWallets          = "/api/v1/wallets"
	RouteWallet           = "/api/v1/wallets/{id}"
	RouteCurrencies       = "/api/v1/currencies"
	RouteCurrency         = "/api/v1/currencies/{id}"
	RouteCurrencyByCode   = "/api/v1/currencies/code/{code}"
	RouteBalances         = "/api/v1/balances"
	RouteBalance          = "/api/v1/balances/{id}"
	RouteTransactions     = "/api/v1/transactions"
	RouteTransaction      = "/api/v1/transactions/{id}"
	RouteAdminLogin       = "/api/v1/admin/login"
	RouteAdminProviders   = "/api/v1/admin/providers"
	RouteAdminProvider    = "/api/v1/admin/providers/{id}"
	RouteAdminWallets     = "/api/v1/admin/wallets"
	RouteAdminWallet      = "/api/v1/admin/wallets/{id}"
	RouteAdminWalletRegen = "/api/v1/admin/wallets/{id}/regenerate"
	RouteAdminWalletIssue = "/api/v1/admin/wallets/{id}/issue-currency"
	RouteAdminCurrencies  = "/api/v1/admin/currencies"
	RouteAdminCurrency    = "/api/v1/admin/currencies/{id}"
	RouteSignInPage       = "/signin"
	RouteWalletsPage      = "/wallets"
	RouteHealth           = "/health"
)

func RegisterRoutes(r chi.Router, pool *pgxpool.Pool, cfg *config.Config) {
	walletSvc := services.NewWalletService(pool)
	currencySvc := services.NewCurrencyService(pool)
	transactionSvc := services.NewTransactionService(pool)
	balanceSvc := services.NewBalanceService(pool)
	authSvc := services.NewAuthService(pool)
	adminSvc := services.NewAdminService(pool, cfg.AdminPassword, cfg.AdminPasswordHash)

	walletHandler := NewWalletHandler(walletSvc)
	currencyHandler := NewCurrencyHandler(currencySvc)
	transactionHandler := NewTransactionHandler(transactionSvc)
	balanceHandler := NewBalanceHandler(balanceSvc)
	authHandler := NewAuthHandler(authSvc)
	adminAPIHandler := NewAdminAPIHandler(adminSvc)
	adminWebHandler := NewAdminWebHandler(adminSvc)
	webHandler := NewWebHandler()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(authMiddleware.RateLimitMiddleware(120, time.Minute))

	registerPublicRoutes(r, webHandler, adminWebHandler, adminAPIHandler, authHandler)
	registerAPIKeyProtectedRoutes(r, pool, authHandler)
	registerAccessTokenProtectedRoutes(r, pool, walletHandler, currencyHandler,
		balanceHandler, transactionHandler)
	registerAdminProtectedRoutes(r, pool, adminAPIHandler)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
}

func registerPublicRoutes(r chi.Router, webHandler *WebHandler,
	adminWebHandler *AdminWebHandler, adminAPIHandler *AdminAPIHandler, authHandler *AuthHandler) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, RouteSignInPage, http.StatusFound)
	})
	r.Get(RouteSignInPage, webHandler.SignInPage)
	r.Get(RouteWalletsPage, webHandler.WalletsPage)
	r.Get(RouteHealth, healthHandler)
	r.Post(RouteSignIn, authHandler.SignIn)
	r.Post(RouteAdminLogin, adminAPIHandler.Login)

	r.Get("/admin/login", adminWebHandler.LoginPage)
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
	})
	r.Get("/admin/dashboard", adminWebHandler.DashboardPage)
	r.Get("/admin/providers", adminWebHandler.ProvidersPage)
	r.Get("/admin/providers/new", adminWebHandler.ProviderCreatePage)
	r.Get("/admin/providers/{id}/edit-api-key", adminWebHandler.ProviderEditPage)
	r.Get("/admin/wallets", adminWebHandler.WalletsPage)
	r.Get("/admin/wallets/new", adminWebHandler.WalletCreatePage)
	r.Get("/admin/wallets/{id}/regenerate", adminWebHandler.WalletRegeneratePage)
	r.Get("/admin/wallets/{id}/issue-currency", adminWebHandler.WalletIssueCurrencyPage)
	r.Get("/admin/currencies", adminWebHandler.CurrenciesPage)
	r.Get("/admin/currencies/new", adminWebHandler.CurrencyCreatePage)
}

func registerAPIKeyProtectedRoutes(r chi.Router, pool *pgxpool.Pool, authHandler *AuthHandler) {
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.APIKeyMiddleware(pool))
		r.Post(RouteRefCodeGen, authHandler.GenerateReferenceCode)
		r.Post(RouteSignUp, authHandler.SignUp)
	})
}

func registerAccessTokenProtectedRoutes(r chi.Router, pool *pgxpool.Pool,
	walletHandler *WalletHandler,
	currencyHandler *CurrencyHandler,
	balanceHandler *BalanceHandler,
	transactionHandler *TransactionHandler) {
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.AccessTokenMiddleware(pool))

		r.Get(RouteWallets, walletHandler.List)
		r.Get(RouteWallet, walletHandler.Get)

		r.Get(RouteCurrencies, currencyHandler.List)
		r.Get(RouteCurrency, currencyHandler.Get)
		r.Get(RouteCurrencyByCode, currencyHandler.GetByCode)

		r.Get(RouteBalances, balanceHandler.List)
		r.Get(RouteBalance, balanceHandler.Get)

		r.Get(RouteTransactions, transactionHandler.List)
		r.Get(RouteTransaction, transactionHandler.Get)
	})
}

func registerAdminProtectedRoutes(r chi.Router, pool *pgxpool.Pool, adminHandler *AdminAPIHandler) {
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(authMiddleware.AdminAuthMiddleware)

		r.Post("/providers", adminHandler.CreateProvider)
		r.Get("/providers", adminHandler.ListProviders)
		r.Put("/providers/{id}", adminHandler.UpdateProvider)
		r.Delete("/providers/{id}", adminHandler.DeleteProvider)

		r.Post("/wallets", adminHandler.CreateWallet)
		r.Get("/wallets", adminHandler.ListWallets)
		r.Post("/wallets/{id}/regenerate", adminHandler.RegenerateWalletPassphrase)
		r.Delete("/wallets/{id}", adminHandler.DeleteWallet)
		r.Post("/wallets/{id}/issue-currency", adminHandler.IssueCurrencyToWallet)
		r.Get("/wallets/{id}/balances", adminHandler.GetWalletBalances)

		r.Post("/currencies", adminHandler.CreateCurrency)
		r.Get("/currencies", adminHandler.ListCurrencies)
		r.Delete("/currencies/{id}", adminHandler.DeleteCurrency)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	common.WriteJSONResponse(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
