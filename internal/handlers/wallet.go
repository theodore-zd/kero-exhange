package handlers

import (
	"net/http"

	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type WalletHandler struct {
	svc *services.WalletService
}

func NewWalletHandler(svc *services.WalletService) *WalletHandler {
	return &WalletHandler{svc: svc}
}

type WalletResponse struct {
	UUID      string `json:"uuid"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type WalletListResponse struct {
	Data []WalletResponse `json:"data"`
	Meta PaginationMeta   `json:"meta"`
}

func (h *WalletHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid wallet UUID")
	if !ok {
		return
	}

	wallet, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	if wallet == nil {
		handleNotFoundError(w, "Wallet")
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, toWalletResponse(wallet))
}

func (h *WalletHandler) List(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	result, err := h.svc.GetAll(r.Context(), params)
	if err != nil {
		common.LogError("WalletHandler.List failed", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list wallets", nil)
		return
	}

	wallets := make([]WalletResponse, len(result.Data))
	for i, w := range result.Data {
		wallets[i] = toWalletResponse(&w)
	}

	common.WriteJSONResponse(w, http.StatusOK, WalletListResponse{
		Data: wallets,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func toWalletResponse(w *db.Wallet) WalletResponse {
	return WalletResponse{
		UUID:      w.UUID.String(),
		CreatedAt: w.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: w.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
