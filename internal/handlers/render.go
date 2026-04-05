package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	common "github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/grove/pkg/grove"
)

var groveEngine *grove.Engine

func InitGrove() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	templatesDir := filepath.Join(dir, "templates")
	store := grove.NewFileSystemStore(templatesDir)
	groveEngine = grove.New(
		grove.WithStore(store),
		grove.WithCacheSize(256),
	)

	return nil
}

func renderGrove(w http.ResponseWriter, r *http.Request, template string, data grove.Data) {
	if data == nil {
		data = grove.Data{}
	}

	result, err := groveEngine.Render(r.Context(), template, data)
	if err != nil {
		common.LogError("template render error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(result.Body))
}
