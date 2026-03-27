package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/middleware/context"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type SignUpRequest struct {
	ReferenceCode string `json:"reference_code"`
}

type SignUpResponse struct {
	AccessToken      string `json:"access_token"`
	WalletUUID       string `json:"wallet_uuid"`
	SecretPassphrase string `json:"secret_passphrase"`
}

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	_, ok := context.GetProviderUUID(r.Context())
	if !ok {
		common.WriteJSONError(w, http.StatusUnauthorized, "MISSING_API_KEY", "Missing X-API-Key header", nil)
		return
	}

	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.ReferenceCode == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_REFERENCE_CODE", "Reference code is required", nil)
		return
	}

	accessToken, walletUUID, passphrase, err := h.svc.SignUp(r.Context(), req.ReferenceCode)
	if err != nil {
		switch err {
		case services.ErrInvalidReferenceCode:
			common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REFERENCE_CODE", "Invalid reference code", nil)
		case services.ErrReferenceCodeUsed:
			common.WriteJSONError(w, http.StatusBadRequest, "REFERENCE_CODE_USED", "Reference code already used", nil)
		case services.ErrReferenceCodeExpired:
			common.WriteJSONError(w, http.StatusBadRequest, "REFERENCE_CODE_EXPIRED", "Reference code expired", nil)
		default:
			common.LogError("SignUp failed", "reference_code", req.ReferenceCode, "error", err)
			common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create wallet", nil)
		}
		return
	}

	common.WriteJSONResponse(w, http.StatusCreated, SignUpResponse{
		AccessToken:      accessToken,
		WalletUUID:       walletUUID,
		SecretPassphrase: passphrase,
	})
}

type SignInRequest struct {
	Passphrase string `json:"passphrase"`
}

type SignInResponse struct {
	AccessToken string `json:"access_token"`
	WalletUUID  string `json:"wallet_uuid"`
}

func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if req.Passphrase == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "MISSING_PASSPHRASE", "Passphrase is required", nil)
		return
	}

	accessToken, walletUUID, err := h.svc.SignIn(r.Context(), req.Passphrase)
	if err != nil {
		if err == services.ErrWalletNotFound {
			common.WriteJSONError(w, http.StatusNotFound, "WALLET_NOT_FOUND", "Wallet not found", nil)
			return
		}
		common.LogError("SignIn failed", "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to sign in", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusOK, SignInResponse{
		AccessToken: accessToken,
		WalletUUID:  walletUUID,
	})
}

type ReferenceCodeResponse struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
}

func (h *AuthHandler) GenerateReferenceCode(w http.ResponseWriter, r *http.Request) {
	providerUUID, ok := context.GetProviderUUID(r.Context())
	if !ok {
		common.WriteJSONError(w, http.StatusUnauthorized, "MISSING_API_KEY", "Missing X-API-Key header", nil)
		return
	}

	rc, err := h.svc.GenerateReferenceCode(r.Context(), providerUUID)
	if err != nil {
		common.LogError("GenerateReferenceCode failed", "provider_uuid", providerUUID, "error", err)
		common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate reference code", nil)
		return
	}

	common.WriteJSONResponse(w, http.StatusCreated, ReferenceCodeResponse{
		Code:      rc.Code,
		ExpiresAt: rc.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}
