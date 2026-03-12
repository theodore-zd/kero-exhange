package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
	UUID            uuid.UUID `json:"uuid"`
	PublicKey       string    `json:"public_key"`
	AccessTokenHash *string   `json:"access_token_hash,omitempty"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

type WalletListResponse struct {
	Data []WalletResponse `json:"data"`
	Meta PaginationMeta   `json:"meta"`
}

func (h *WalletHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/wallets", h.List)
	r.Get("/api/v1/wallets/{id}", h.Get)
}

func (h *WalletHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_UUID", "Invalid wallet UUID", nil)
		return
	}

	wallet, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get wallet", nil)
		return
	}

	if wallet == nil {
		common.WriteJSONError(w, http.StatusNotFound, "NOT_FOUND", "Wallet not found", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, toWalletResponse(wallet))
}

func (h *WalletHandler) List(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	filter := db.WalletFilter{}
	if pk := common.ParseQueryStringPtr(r, "public_key"); pk != nil {
		filter.PublicKey = *pk
	}

	result, err := h.svc.GetAll(r.Context(), params, filter)
	if err != nil {
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
		UUID:            w.UUID,
		PublicKey:       w.PublicKey,
		AccessTokenHash: w.AccessTokenHash,
		CreatedAt:       w.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       w.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func init() {
	var _ json.Marshaler = (*PaginationMeta)(nil)
}
