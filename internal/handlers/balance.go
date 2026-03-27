package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type BalanceHandler struct {
	svc *services.BalanceService
}

func NewBalanceHandler(svc *services.BalanceService) *BalanceHandler {
	return &BalanceHandler{svc: svc}
}

type BalanceResponse struct {
	UUID         uuid.UUID `json:"uuid"`
	WalletID     uuid.UUID `json:"wallet_id"`
	CurrencyID   uuid.UUID `json:"currency_id"`
	Balance      string    `json:"balance"`
	UpdatedAt    string    `json:"updated_at"`
	CurrencyCode string    `json:"currency_code"`
	CurrencyName string    `json:"currency_name"`
}

type BalanceListResponse struct {
	Data []BalanceResponse `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

func (h *BalanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid balance UUID")
	if !ok {
		return
	}

	balance, err := h.svc.GetByUUID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	if balance == nil {
		handleNotFoundError(w, "Balance")
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, toBalanceResponse(balance))
}

func (h *BalanceHandler) List(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	filter := db.BalanceFilter{}
	if walletIDStr := common.ParseQueryStringPtr(r, "wallet_id"); walletIDStr != nil {
		walletID, err := uuid.Parse(*walletIDStr)
		if err == nil {
			filter.WalletID = walletID
		}
	}

	if currencyIDStr := common.ParseQueryStringPtr(r, "currency_id"); currencyIDStr != nil {
		currencyID, err := uuid.Parse(*currencyIDStr)
		if err == nil {
			filter.CurrencyID = currencyID
		}
	}

	result, err := h.svc.GetAll(r.Context(), params, filter)
	if err != nil {
		common.LogError("BalanceHandler.List failed", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list balances", nil)
		return
	}

	balances := make([]BalanceResponse, len(result.Data))
	for i, b := range result.Data {
		balances[i] = toBalanceResponse(&b)
	}

	common.WriteJSONResponse(w, http.StatusOK, BalanceListResponse{
		Data: balances,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func toBalanceResponse(b *db.Balance) BalanceResponse {
	return BalanceResponse{
		UUID:         b.UUID,
		WalletID:     b.WalletID,
		CurrencyID:   b.CurrencyID,
		Balance:      b.Balance.String(),
		UpdatedAt:    b.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		CurrencyCode: b.CurrencyCode,
		CurrencyName: b.CurrencyName,
	}
}
