package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wispberry-tech/grove/pkg/grove"
)

type AdminWebHandler struct{}

func NewAdminWebHandler() *AdminWebHandler {
	return &AdminWebHandler{}
}

func (h *AdminWebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/login.grov", nil)
}

func (h *AdminWebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/dashboard.grov", grove.Data{"active": "dashboard"})
}

func (h *AdminWebHandler) ProvidersPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/providers.grov", grove.Data{"active": "providers"})
}

func (h *AdminWebHandler) ProviderCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/provider-form.grov", grove.Data{"mode": "create", "active": "providers"})
}

func (h *AdminWebHandler) ProviderEditPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/provider-form.grov", grove.Data{"mode": "edit", "provider_id": id, "active": "providers"})
}

func (h *AdminWebHandler) WalletsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/wallets.grov", grove.Data{"active": "wallets"})
}

func (h *AdminWebHandler) WalletCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "create", "active": "wallets"})
}

func (h *AdminWebHandler) WalletDetailPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-detail.grov", grove.Data{"wallet_id": id, "active": "wallets"})
}

func (h *AdminWebHandler) WalletRegeneratePage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "regenerate", "wallet_id": id, "active": "wallets"})
}

func (h *AdminWebHandler) WalletIssueCurrencyPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "issue", "wallet_id": id, "active": "wallets"})
}

func (h *AdminWebHandler) CurrenciesPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/currencies.grov", grove.Data{"active": "currencies"})
}

func (h *AdminWebHandler) CurrencyCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/currency-form.grov", grove.Data{"active": "currencies"})
}

func (h *AdminWebHandler) TransactionsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/transactions.grov", grove.Data{"active": "transactions"})
}

func (h *AdminWebHandler) AuditLogPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/audit-log.grov", grove.Data{"active": "audit-log"})
}

func (h *AdminWebHandler) LookupPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/user-lookup.grov", grove.Data{"active": "lookup"})
}
