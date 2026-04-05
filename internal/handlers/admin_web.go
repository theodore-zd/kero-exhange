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
	renderGrove(w, r, "admin/dashboard.grov", nil)
}

func (h *AdminWebHandler) ProvidersPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/providers.grov", nil)
}

func (h *AdminWebHandler) ProviderCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/provider-form.grov", grove.Data{"mode": "create"})
}

func (h *AdminWebHandler) ProviderEditPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/provider-form.grov", grove.Data{"mode": "edit", "provider_id": id})
}

func (h *AdminWebHandler) WalletsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/wallets.grov", nil)
}

func (h *AdminWebHandler) WalletCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "create"})
}

func (h *AdminWebHandler) WalletDetailPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-detail.grov", grove.Data{"wallet_id": id})
}

func (h *AdminWebHandler) WalletRegeneratePage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "regenerate", "wallet_id": id})
}

func (h *AdminWebHandler) WalletIssueCurrencyPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	renderGrove(w, r, "admin/wallet-form.grov", grove.Data{"mode": "issue", "wallet_id": id})
}

func (h *AdminWebHandler) CurrenciesPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/currencies.grov", nil)
}

func (h *AdminWebHandler) CurrencyCreatePage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/currency-form.grov", nil)
}

func (h *AdminWebHandler) TransactionsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/transactions.grov", nil)
}

func (h *AdminWebHandler) AuditLogPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/audit-log.grov", nil)
}

func (h *AdminWebHandler) LookupPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "admin/user-lookup.grov", nil)
}
