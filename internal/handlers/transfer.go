package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	mctx "github.com/wispberry-tech/kero-exchange/internal/middleware/context"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type TransferHandler struct {
	svc *services.TransferService
}

func NewTransferHandler(svc *services.TransferService) *TransferHandler {
	return &TransferHandler{svc: svc}
}

type TransferRequest struct {
	DestinationWalletID string `json:"destination_wallet_id"`
	CurrencyID          string `json:"currency_id"`
	Amount              string `json:"amount"`
}

type TransferTransactionResponse struct {
	UUID       uuid.UUID `json:"uuid"`
	WalletID   uuid.UUID `json:"wallet_id"`
	CurrencyID uuid.UUID `json:"currency_id"`
	Amount     string    `json:"amount"`
	Type       string    `json:"type"`
	Timestamp  string    `json:"timestamp"`
}

type TransferResponse struct {
	TransferID uuid.UUID                   `json:"transfer_id"`
	Debit      TransferTransactionResponse `json:"debit"`
	Credit     TransferTransactionResponse `json:"credit"`
}

func (h *TransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	sourceWalletID, ok := mctx.GetWalletUUID(r.Context())
	if !ok {
		common.WriteJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication", nil)
		return
	}

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.DestinationWalletID == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_DESTINATION", "destination_wallet_id is required", nil)
		return
	}

	destWalletID, err := uuid.Parse(req.DestinationWalletID)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_DESTINATION", "Invalid destination_wallet_id UUID", nil)
		return
	}

	if req.CurrencyID == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_CURRENCY", "currency_id is required", nil)
		return
	}

	currencyID, err := uuid.Parse(req.CurrencyID)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_CURRENCY", "Invalid currency_id UUID", nil)
		return
	}

	if req.Amount == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_AMOUNT", "amount is required", nil)
		return
	}

	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_AMOUNT", "Invalid amount", nil)
		return
	}

	result, err := h.svc.Transfer(r.Context(), services.TransferParams{
		SourceWalletID: sourceWalletID,
		DestWalletID:   destWalletID,
		CurrencyID:     currencyID,
		Amount:         amount,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSameWallet):
			common.WriteJSONError(w, http.StatusBadRequest, "SAME_WALLET", "Source and destination wallet cannot be the same", nil)
		case errors.Is(err, services.ErrInsufficientBalance):
			common.WriteJSONError(w, http.StatusBadRequest, "INSUFFICIENT_BALANCE", "Insufficient balance for transfer", nil)
		case errors.Is(err, services.ErrDestinationNotFound):
			common.WriteJSONError(w, http.StatusBadRequest, "DESTINATION_NOT_FOUND", "Destination wallet not found", nil)
		case errors.Is(err, services.ErrCurrencyNotFound):
			common.WriteJSONError(w, http.StatusBadRequest, "CURRENCY_NOT_FOUND", "Currency not found", nil)
		default:
			common.LogError("TransferHandler.Create failed", "error", err)
			common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process transfer", nil)
		}
		return
	}

	common.WriteJSONResponse(w, http.StatusCreated, TransferResponse{
		TransferID: result.TransferID,
		Debit:      toTransferTransactionResponse(result.Debit),
		Credit:     toTransferTransactionResponse(result.Credit),
	})
}

func toTransferTransactionResponse(t *db.Transaction) TransferTransactionResponse {
	return TransferTransactionResponse{
		UUID:       t.UUID,
		WalletID:   t.WalletID,
		CurrencyID: t.CurrencyID,
		Amount:     t.Amount.String(),
		Type:       string(t.Type),
		Timestamp:  t.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
