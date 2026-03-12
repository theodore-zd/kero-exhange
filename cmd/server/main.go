package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wispberry-tech/go-common"
	"github.com/wispberry-tech/kero-exchange/internal/config"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/handlers"
)

func main() {
	common.InitializeLogger()

	cfg, err := config.Load()
	if err != nil {
		common.LogError("Failed to load config", "error", err)
		os.Exit(1)
	}

	common.SetLogLevel(cfg.LogLevel)
	common.LogInfo("Starting kero-exchange server", "version", "0.1.0")

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		common.LogError("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	common.LogInfo("Connected to database")

	r := chi.NewRouter()
	handlers.RegisterRoutes(r, pool)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		common.LogInfo("Server listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			common.LogError("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	common.LogInfo("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		common.LogError("Server shutdown error", "error", err)
	}

	common.LogInfo("Server stopped")
}
