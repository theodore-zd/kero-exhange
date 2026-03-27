package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/wispberry-tech/go-common"
)

func parseUUID(r *http.Request, paramName string) (uuid.UUID, error) {
	idStr := chi.URLParam(r, paramName)
	return uuid.Parse(idStr)
}

func parseUUIDOrError(w http.ResponseWriter, r *http.Request, paramName string, errorCode string, errorMsg string) (uuid.UUID, bool) {
	id, err := parseUUID(r, paramName)
	if err != nil {
		common.WriteJSONError(w, http.StatusBadRequest, errorCode, errorMsg, nil)
		return uuid.Nil, false
	}
	return id, true
}

func handleNotFoundError(w http.ResponseWriter, resourceType string) {
	common.WriteJSONError(w, http.StatusNotFound, "NOT_FOUND", resourceType+" not found", nil)
}

func handleServiceError(w http.ResponseWriter, err error) {
	common.LogError("Service error", "error", err)
	common.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
		"An internal error occurred", nil)
}
