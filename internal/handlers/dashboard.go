package handlers

import (
	"net/http"

	"github.com/wispberry-tech/go-common"
	mctx "github.com/wispberry-tech/kero-exchange/internal/middleware/context"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type DashboardHandler struct {
	svc *services.DashboardService
}

func NewDashboardHandler(svc *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

type BalanceSummaryResponse struct {
	CurrencyCode string `json:"currency_code"`
	CurrencyName string `json:"currency_name"`
	TotalAmount  string `json:"total_amount"`
}

type DashboardResponse struct {
	WalletCount int                      `json:"wallet_count"`
	Balances    []BalanceSummaryResponse `json:"balances"`
}

func (h *DashboardHandler) Summary(w http.ResponseWriter, r *http.Request) {
	walletID, ok := mctx.GetWalletUUID(r.Context())
	if !ok {
		common.WriteJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication", nil)
		return
	}

	summary, err := h.svc.GetSummary(r.Context(), walletID)
	if err != nil {
		common.LogError("DashboardHandler.Summary failed", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get dashboard summary", nil)
		return
	}

	balances := make([]BalanceSummaryResponse, len(summary.Balances))
	for i, b := range summary.Balances {
		balances[i] = BalanceSummaryResponse{
			CurrencyCode: b.CurrencyCode,
			CurrencyName: b.CurrencyName,
			TotalAmount:  b.TotalAmount.String(),
		}
	}

	common.WriteJSONResponse(w, http.StatusOK, DashboardResponse{
		WalletCount: summary.WalletCount,
		Balances:    balances,
	})
}
