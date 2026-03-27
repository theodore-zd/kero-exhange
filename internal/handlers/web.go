package handlers

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

type WebHandler struct {
	templatesDir    string
	baseTemplate    string
	standaloneFiles map[string]bool
}

func NewWebHandler() *WebHandler {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	templatesDir := filepath.Join(dir, "templates")

	files, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	standaloneFiles := make(map[string]bool)

	if err == nil && len(files) > 0 {
		for _, file := range files {
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

	return &WebHandler{templatesDir: templatesDir, baseTemplate: filepath.Join(templatesDir, "base.html"), standaloneFiles: standaloneFiles}
}

func (h *WebHandler) SignInPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "signin.html", map[string]bool{"ShowSidebar": false})
}

func (h *WebHandler) WalletsPage(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, "wallets.html", map[string]bool{"ShowSidebar": true})
}

func (h *WebHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
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
