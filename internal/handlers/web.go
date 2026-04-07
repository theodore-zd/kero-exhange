package handlers

import (
	"net/http"
)

type WebHandler struct{}

func NewWebHandler() *WebHandler {
	return &WebHandler{}
}

func (h *WebHandler) SignInPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/signin.grov", nil)
}

func (h *WebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	renderGrove(w, r, "user/dashboard.grov", nil)
}
