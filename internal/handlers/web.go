package handlers

import (
	"net/http"
)

type WebHandler struct{}

func NewWebHandler() *WebHandler {
	return &WebHandler{}
}

func (h *WebHandler) SignInPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/signin.grov", nil)
}

func (h *WebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/dashboard.grov", nil)
}

func (h *WebHandler) WalletsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/wallets.grov", nil)
}

func (h *WebHandler) WalletDetailPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/wallet-detail.grov", nil)
}

func (h *WebHandler) TransactionsPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/transactions.grov", nil)
}

func (h *WebHandler) TransferPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/transfer.grov", nil)
}
