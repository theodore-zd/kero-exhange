package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wispberry-tech/grove/pkg/grove"
)

type WebHandler struct{}

func NewWebHandler() *WebHandler {
	return &WebHandler{}
}

func (h *WebHandler) SignInPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/signin.grov", nil)
}

func (h *WebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/dashboard.grov", grove.Data{"active": "dashboard"})
}

func (h *WebHandler) WalletsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/wallets.grov", grove.Data{"active": "wallets"})
}

func (h *WebHandler) WalletDetailPage(w http.ResponseWriter, r *http.Request) {
	walletID := chi.URLParam(r, "id")
	renderGrove(w, r, "user/wallet-detail.grov", grove.Data{"active": "wallets", "wallet_id": walletID})
}

func (h *WebHandler) TransactionsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/transactions.grov", grove.Data{"active": "transactions"})
}

func (h *WebHandler) TransferPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/transfer.grov", grove.Data{"active": "transfer"})
}
