package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	authMiddleware "github.com/wispberry-tech/kero-exchange/internal/middleware"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type AdminAPIHandler struct {
	svc *services.AdminService
}

func NewAdminAPIHandler(svc *services.AdminService) *AdminAPIHandler {
	return &AdminAPIHandler{svc: svc}
}

func (h *AdminAPIHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AdminLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Password == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_PASSWORD", "Password is required", nil)
		return
	}

	if !h.svc.CheckAdminPassword(req.Password) {
		common.WriteJSONError(w, http.StatusUnauthorized, "INVALID_PASSWORD", "Invalid admin password", nil)
		return
	}

	token := h.svc.GenerateAdminToken()
	if token == "" {
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate admin token", nil)
		return
	}
	authMiddleware.StoreAdminToken(token, 24*time.Hour)
	common.WriteJSONResponse(w, http.StatusOK, AdminLoginResponse{Token: token})
}

func (h *AdminAPIHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req CreateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Name == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_NAME", "Provider name is required", nil)
		return
	}

	provider, apiKey, err := h.svc.CreateProvider(r.Context(), req.Name, "admin", getClientIP(r), r.UserAgent())
	if err != nil {
		common.LogError("CreateProvider failed", "name", req.Name, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create provider", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusCreated, CreateProviderResponse{
		UUID:       provider.UUID.String(),
		APIKey:     apiKey,
		APIKeyHash: provider.APIKeyHash,
		Name:       provider.Name,
		CreatedAt:  provider.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

func (h *AdminAPIHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid provider UUID")
	if !ok {
		return
	}

	var req UpdateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.APIKey == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_API_KEY", "API key is required", nil)
		return
	}

	if err := h.svc.UpdateProviderAPIKey(r.Context(), id, req.APIKey, "admin", getClientIP(r), r.UserAgent()); err != nil {
		common.LogError("UpdateProviderAPIKey failed", "provider_id", id, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update provider", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminAPIHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	result, err := h.svc.ListProviders(r.Context(), params)
	if err != nil {
		common.LogError("ListProviders failed", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list providers", nil)
		return
	}

	providers := make([]ProviderResponse, len(result.Data))
	for i, p := range result.Data {
		providers[i] = ProviderResponse{
			UUID:       p.UUID.String(),
			APIKeyHash: p.APIKeyHash,
			Name:       p.Name,
			CreatedAt:  p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	common.WriteJSONResponse(w, http.StatusOK, ProviderListResponse{
		Data: providers,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func (h *AdminAPIHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid provider UUID")
	if !ok {
		return
	}

	if err := h.svc.DeleteProvider(r.Context(), id, "admin", getClientIP(r), r.UserAgent()); err != nil {
		common.LogError("DeleteProvider failed", "provider_id", id, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete provider", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminAPIHandler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	wallet, passphrase, accessToken, err := h.svc.CreateWallet(r.Context(), "admin", getClientIP(r), r.UserAgent())
	if err != nil {
		common.LogError("CreateWallet failed", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create wallet", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusCreated, CreateWalletResponse{
		UUID:        wallet.UUID.String(),
		Passphrase:  passphrase,
		AccessToken: accessToken,
		CreatedAt:   wallet.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   wallet.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func (h *AdminAPIHandler) RegenerateWalletPassphrase(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid wallet UUID")
	if !ok {
		return
	}

	wallet, passphrase, accessToken, err := h.svc.RegenerateWalletPassphrase(r.Context(), id, "admin", getClientIP(r), r.UserAgent())
	if err != nil {
		common.LogError("RegenerateWalletPassphrase failed", "wallet_id", id, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to regenerate passphrase", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, RegenerateWalletResponse{
		UUID:        wallet.UUID.String(),
		Passphrase:  passphrase,
		AccessToken: accessToken,
		CreatedAt:   wallet.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   wallet.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func (h *AdminAPIHandler) ListWallets(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	result, err := h.svc.ListWallets(r.Context(), params)
	if err != nil {
		common.LogError("ListWallets failed", "error", err)
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

func (h *AdminAPIHandler) DeleteWallet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid wallet UUID")
	if !ok {
		return
	}

	if err := h.svc.DeleteWallet(r.Context(), id, "admin", getClientIP(r), r.UserAgent()); err != nil {
		common.LogError("DeleteWallet failed", "wallet_id", id, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete wallet", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminAPIHandler) CreateCurrency(w http.ResponseWriter, r *http.Request) {
	var req CreateCurrencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Code == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_CODE", "Currency code is required", nil)
		return
	}

	if req.Name == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_NAME", "Currency name is required", nil)
		return
	}

	if len(req.Code) > 8 {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_CODE", "Currency code must be at most 8 characters", nil)
		return
	}

	currency, err := h.svc.CreateCurrency(r.Context(), services.CreateCurrencyParams{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
	}, "admin", getClientIP(r), r.UserAgent())
	if err != nil {
		common.LogError("CreateCurrency failed", "code", req.Code, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create currency", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusCreated, CurrencyAdminResponse{
		UUID:        currency.UUID.String(),
		Code:        currency.Code,
		Name:        currency.Name,
		Description: currency.Description,
		CreatedAt:   currency.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func (h *AdminAPIHandler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	result, err := h.svc.GetAllCurrencies(r.Context(), params)
	if err != nil {
		common.LogError("ListCurrencies failed", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list currencies", nil)
		return
	}

	currencies := make([]CurrencyAdminResponse, len(result.Data))
	for i, c := range result.Data {
		currencies[i] = CurrencyAdminResponse{
			UUID:        c.UUID.String(),
			Code:        c.Code,
			Name:        c.Name,
			Description: c.Description,
			CreatedAt:   c.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
	}

	common.WriteJSONResponse(w, http.StatusOK, CurrencyListAdminResponse{
		Data: currencies,
		Meta: PaginationMeta{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

func (h *AdminAPIHandler) DeleteCurrency(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid currency UUID")
	if !ok {
		return
	}

	if err := h.svc.DeleteCurrency(r.Context(), id, "admin", getClientIP(r), r.UserAgent()); err != nil {
		common.LogError("DeleteCurrency handler failed", "currency_id", id, "error", err)
		errMsg := err.Error()
		if strings.Contains(errMsg, "cannot delete currency") {
			common.WriteJSONError(w, http.StatusConflict, "CURRENCY_IN_USE", errMsg, nil)
			return
		}
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete currency", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminAPIHandler) GetWalletBalances(w http.ResponseWriter, r *http.Request) {
	walletID, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid wallet UUID")
	if !ok {
		return
	}

	params := db.PaginationParams{
		Page:     common.ParseQueryInt(r, "page", 1),
		PageSize: common.ParseQueryInt(r, "page_size", 20),
	}

	result, err := h.svc.GetWalletBalances(r.Context(), walletID, params)
	if err != nil {
		common.LogError("GetWalletBalances failed", "wallet_id", walletID, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get wallet balances", nil)
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

func (h *AdminAPIHandler) IssueCurrencyToWallet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUIDOrError(w, r, "id", "INVALID_UUID", "Invalid wallet UUID")
	if !ok {
		return
	}

	var req IssueCurrencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.CurrencyID == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_CURRENCY_ID", "Currency ID is required", nil)
		return
	}

	if req.Amount == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_AMOUNT", "Amount is required", nil)
		return
	}

	currencyID, err := uuid.Parse(req.CurrencyID)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_CURRENCY_ID", "Invalid currency UUID", nil)
		return
	}

	result, err := h.svc.IssueCurrency(r.Context(), services.IssueCurrencyAdminParams{
		WalletID:   id,
		CurrencyID: currencyID,
		Amount:     req.Amount,
		Reference:  req.Reference,
	}, "admin", getClientIP(r), r.UserAgent())
	if err != nil {
		common.LogError("IssueCurrencyToWallet failed", "wallet_id", id, "currency_id", currencyID, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to issue currency", nil)
		return
	}

	balanceResp := toBalanceResponse(result.Balance)
	common.WriteJSONResponse(w, http.StatusCreated, IssueCurrencyResponse{
		Balance:     &balanceResp,
		Transaction: toTransactionResponse(result.Transaction),
		WalletUUID:  id.String(),
	})
}
