package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/wispberry-tech/kero-exchange/internal/config"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

var rootCmd = &cobra.Command{
	Use:   "kero",
	Short: "Kero Exchange Admin CLI",
	Long:  "Administrative CLI for managing currencies and issuing funds to wallets.",
}

var (
	pool            *pgxpool.Pool
	currencyService *services.CurrencyService
	walletService   *services.WalletService
	balanceService  *services.BalanceService
)

func main() {
	cfg, err := config.LoadForCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err = db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	currencyService = services.NewCurrencyService(pool)
	walletService = services.NewWalletService(pool)
	balanceService = services.NewBalanceService(pool)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
