package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Manage wallet balances",
}

var balanceIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue currency to a wallet (admin operation)",
	Long:  "Creates or updates a balance and records an admin_issued transaction.",
	Run:   runBalanceIssue,
}

var balanceGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get wallet balances",
	Run:   runBalanceGet,
}

var (
	issueWalletID   string
	issueCurrencyID string
	issueAmount     string
	issueReference  string
	getWalletID     string
)

func init() {
	balanceIssueCmd.Flags().StringVarP(&issueWalletID, "wallet", "w", "", "Wallet UUID")
	balanceIssueCmd.Flags().StringVarP(&issueCurrencyID, "currency", "c", "", "Currency UUID or code")
	balanceIssueCmd.Flags().StringVarP(&issueAmount, "amount", "a", "", "Amount to issue")
	balanceIssueCmd.Flags().StringVarP(&issueReference, "reference", "r", "", "Optional reference note")
	balanceIssueCmd.MarkFlagRequired("wallet")
	balanceIssueCmd.MarkFlagRequired("currency")
	balanceIssueCmd.MarkFlagRequired("amount")

	balanceGetCmd.Flags().StringVarP(&getWalletID, "wallet", "w", "", "Wallet UUID")
	balanceGetCmd.MarkFlagRequired("wallet")

	balanceCmd.AddCommand(balanceIssueCmd)
	balanceCmd.AddCommand(balanceGetCmd)
	rootCmd.AddCommand(balanceCmd)
}

func runBalanceIssue(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	walletUUID, err := uuid.Parse(issueWalletID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid wallet UUID: %v\n", err)
		os.Exit(1)
	}

	var currencyUUID uuid.UUID
	if parsed, err := uuid.Parse(issueCurrencyID); err == nil {
		currencyUUID = parsed
	} else {
		currency, err := currencyService.GetByCode(ctx, issueCurrencyID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error looking up currency: %v\n", err)
			os.Exit(1)
		}
		if currency == nil {
			fmt.Fprintf(os.Stderr, "Currency not found: %s\n", issueCurrencyID)
			os.Exit(1)
		}
		currencyUUID = currency.UUID
	}

	amount, err := decimal.NewFromString(issueAmount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid amount: %v\n", err)
		os.Exit(1)
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		fmt.Fprintln(os.Stderr, "Amount must be positive")
		os.Exit(1)
	}

	var reference *string
	if issueReference != "" {
		reference = &issueReference
	}

	result, err := balanceService.IssueCurrency(ctx, services.IssueCurrencyParams{
		WalletID:   walletUUID,
		CurrencyID: currencyUUID,
		Amount:     amount,
		Reference:  reference,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error issuing currency: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Currency issued successfully:\n")
	fmt.Printf("  Wallet: %s\n", walletUUID)
	fmt.Printf("  Currency: %s\n", currencyUUID)
	fmt.Printf("  Amount: %s\n", amount.String())
	fmt.Printf("  New Balance: %s\n", result.Balance.Balance.String())
	fmt.Printf("  Transaction: %s\n", result.Transaction.UUID)
}

func runBalanceGet(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	walletUUID, err := uuid.Parse(getWalletID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid wallet UUID: %v\n", err)
		os.Exit(1)
	}

	result, err := balanceService.GetAll(ctx, db.PaginationParams{Page: 1, PageSize: 100}, db.BalanceFilter{
		WalletID: walletUUID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting balances: %v\n", err)
		os.Exit(1)
	}

	if len(result.Data) == 0 {
		fmt.Println("No balances found for this wallet.")
		return
	}

	fmt.Printf("Balances for wallet %s:\n", walletUUID)
	for _, b := range result.Data {
		currency, _ := currencyService.GetByID(ctx, b.CurrencyID)
		code := b.CurrencyID.String()
		if currency != nil {
			code = currency.Code
		}
		fmt.Printf("  %s: %s\n", code, b.Balance.String())
	}
}
