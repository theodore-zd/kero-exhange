package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type CurrencyHandler struct {
	svc *services.CurrencyService
}

func NewCurrencyHandler(svc *services.CurrencyService) *CurrencyHandler {
	return &CurrencyHandler{svc: svc}
}

type CurrencyResponse struct {
	UUID        uuid.UUID `json:"uuid"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

type CurrencyListResponse struct {
	Data []CurrencyResponse `json:"data"`
	Meta PaginationMeta     `json:"meta"`
}

func (h *CurrencyHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/currencies", h.List)
	r.Get("/api/v1/currencies/{id}", h.Get)
	r.Get("/api/v1/currencies/code/{code}", h.GetByCode)
}

func (h *CurrencyHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_UUID", "Invalid currency UUID", nil)
		return
	}

	currency, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get currency", nil)
		return
	}

	if currency == nil {
		common.WriteJSONError(w, http.StatusNotFound, "NOT_FOUND", "Currency not found", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, toCurrencyResponse(currency))
}

func (h *CurrencyHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_CODE", "Currency code is required", nil)
		return
	}

	currency, err := h.svc.GetByCode(r.Context(), code)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get currency", nil)
		return
	}

	if currency == nil {
		common.WriteJSONError(w, http.StatusNotFound, "NOT_FOUND", "Currency not found", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, toCurrencyResponse(currency))
}

func (h *CurrencyHandler) List(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	result, err := h.svc.GetAll(r.Context(), params)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list currencies", nil)
		return
	}

	currencies := make([]CurrencyResponse, len(result.Data))
	for i, c := range result.Data {
		currencies[i] = toCurrencyResponse(&c)
	}

	common.WriteJSONResponse(w, http.StatusOK, CurrencyListResponse{
		Data: currencies,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func toCurrencyResponse(c *db.Currency) CurrencyResponse {
	return CurrencyResponse{
		UUID:        c.UUID,
		Code:        c.Code,
		Name:        c.Name,
		Description: c.Description,
		CreatedAt:   c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
