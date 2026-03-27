package handlers

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

type AdminWebHandler struct {
	svc             *services.AdminService
	templatesDir    string
	baseTemplate    string
	standaloneFiles map[string]bool
}

func NewAdminWebHandler(svc *services.AdminService) *AdminWebHandler {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	templatesDir := filepath.Join(dir, "templates", "admin")

	adminFiles, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	standaloneFiles := make(map[string]bool)

	if err == nil && len(adminFiles) > 0 {
		for _, file := range adminFiles {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}
			filename := filepath.Base(file)
			if len(content) > 15 && string(content[:15]) == "<!DOCTYPE html>" {
				standaloneFiles[filename] = true
			}
		}
	}

	return &AdminWebHandler{svc: svc, templatesDir: templatesDir, baseTemplate: filepath.Join(dir, "templates", "base.html"), standaloneFiles: standaloneFiles}
}

func (h *AdminWebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "login.html", map[string]bool{"ShowSidebar": false})
}

func (h *AdminWebHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "dashboard.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) ProvidersPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "providers.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) ProviderCreatePage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "provider_create.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) ProviderEditPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "provider_edit.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) WalletsPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "wallets.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) WalletCreatePage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "wallet_create.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) WalletRegeneratePage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "wallet_regenerate.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) WalletIssueCurrencyPage(w http.ResponseWriter, r *http.Request) {
	walletUUID := chi.URLParam(r, "id")
	if walletUUID == "" {
		http.Error(w, "Wallet ID required", http.StatusBadRequest)
		return
	}
	data := map[string]interface{}{
		"WalletUUID":  walletUUID,
		"ShowSidebar": true,
	}
	h.renderTemplate(w, "wallet_issue_currency.html", data)
}

func (h *AdminWebHandler) CurrenciesPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "currencies.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) CurrencyCreatePage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "currency_create.html", map[string]bool{"ShowSidebar": true})
}

func (h *AdminWebHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if h.templatesDir == "" {
		http.Error(w, "Templates not loaded", http.StatusInternalServerError)
		return
	}

	if h.standaloneFiles[name] {
		tmpl, err := template.ParseFiles(filepath.Join(h.templatesDir, name))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	tmpl, err := template.ParseFiles(h.baseTemplate, filepath.Join(h.templatesDir, name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
