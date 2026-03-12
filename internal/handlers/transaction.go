package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type TransactionHandler struct {
	svc *services.TransactionService
}

func NewTransactionHandler(svc *services.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

type TransactionResponse struct {
	UUID       uuid.UUID `json:"uuid"`
	WalletID   uuid.UUID `json:"wallet_id"`
	CurrencyID uuid.UUID `json:"currency_id"`
	Amount     string    `json:"amount"`
	Type       string    `json:"type"`
	Reference  *string   `json:"reference,omitempty"`
	Timestamp  string    `json:"timestamp"`
}

type TransactionListResponse struct {
	Data []TransactionResponse `json:"data"`
	Meta PaginationMeta        `json:"meta"`
}

func (h *TransactionHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/transactions", h.List)
	r.Get("/api/v1/transactions/{id}", h.Get)
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_UUID", "Invalid transaction UUID", nil)
		return
	}

	tx, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get transaction", nil)
		return
	}

	if tx == nil {
		common.WriteJSONError(w, http.StatusNotFound, "NOT_FOUND", "Transaction not found", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, toTransactionResponse(tx))
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	filter := db.TransactionFilter{}

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

	if txType := common.ParseQueryStringPtr(r, "type"); txType != nil {
		filter.Type = db.TransactionType(*txType)
	}

	if startDate := common.ParseQueryTime(r, "start_date"); !startDate.IsZero() {
		filter.StartDate = &startDate
	}

	if endDate := common.ParseQueryTime(r, "end_date"); !endDate.IsZero() {
		filter.EndDate = &endDate
	}

	result, err := h.svc.GetAll(r.Context(), params, filter)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list transactions", nil)
		return
	}

	transactions := make([]TransactionResponse, len(result.Data))
	for i, t := range result.Data {
		transactions[i] = toTransactionResponse(&t)
	}

	common.WriteJSONResponse(w, http.StatusOK, TransactionListResponse{
		Data: transactions,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func toTransactionResponse(t *db.Transaction) TransactionResponse {
	return TransactionResponse{
		UUID:       t.UUID,
		WalletID:   t.WalletID,
		CurrencyID: t.CurrencyID,
		Amount:     t.Amount.String(),
		Type:       string(t.Type),
		Reference:  t.Reference,
		Timestamp:  t.Timestamp.UTC().Format(time.RFC3339),
	}
}
