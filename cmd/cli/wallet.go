package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Manage wallets",
}

var walletListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all wallets",
	Run:   runWalletList,
}

func init() {
	walletCmd.AddCommand(walletListCmd)
	rootCmd.AddCommand(walletCmd)
}

func runWalletList(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	result, err := walletService.GetAll(ctx, db.PaginationParams{Page: 1, PageSize: 100}, db.WalletFilter{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing wallets: %v\n", err)
		os.Exit(1)
	}

	if len(result.Data) == 0 {
		fmt.Println("No wallets found.")
		return
	}

	fmt.Printf("Wallets (%d total):\n", result.Total)
	for _, w := range result.Data {
		fmt.Printf("  %s | %s\n", w.UUID, w.PublicKey)
	}
}
