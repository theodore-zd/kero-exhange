package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

var currencyCmd = &cobra.Command{
	Use:   "currency",
	Short: "Manage currencies",
}

var currencyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new currency",
	Run:   runCurrencyCreate,
}

var currencyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all currencies",
	Run:   runCurrencyList,
}

var currencyGetCmd = &cobra.Command{
	Use:   "get [uuid|code]",
	Short: "Get currency by UUID or code",
	Args:  cobra.ExactArgs(1),
	Run:   runCurrencyGet,
}

var (
	currencyCode        string
	currencyName        string
	currencyDescription string
)

func init() {
	currencyCreateCmd.Flags().StringVarP(&currencyCode, "code", "c", "", "Currency code (e.g., BTC)")
	currencyCreateCmd.Flags().StringVarP(&currencyName, "name", "n", "", "Currency name (e.g., Bitcoin)")
	currencyCreateCmd.Flags().StringVarP(&currencyDescription, "description", "d", "", "Currency description")
	currencyCreateCmd.MarkFlagRequired("code")
	currencyCreateCmd.MarkFlagRequired("name")

	currencyCmd.AddCommand(currencyCreateCmd)
	currencyCmd.AddCommand(currencyListCmd)
	currencyCmd.AddCommand(currencyGetCmd)
	rootCmd.AddCommand(currencyCmd)
}

func runCurrencyCreate(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	var desc *string
	if currencyDescription != "" {
		desc = &currencyDescription
	}

	currency, err := currencyService.Create(ctx, db.CreateCurrencyParams{
		Code:        currencyCode,
		Name:        currencyName,
		Description: desc,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating currency: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Currency created:\n")
	fmt.Printf("  UUID: %s\n", currency.UUID)
	fmt.Printf("  Code: %s\n", currency.Code)
	fmt.Printf("  Name: %s\n", currency.Name)
	if currency.Description != nil {
		fmt.Printf("  Description: %s\n", *currency.Description)
	}
	fmt.Printf("  Created: %s\n", currency.CreatedAt)
}

func runCurrencyList(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	result, err := currencyService.GetAll(ctx, db.PaginationParams{Page: 1, PageSize: 100})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing currencies: %v\n", err)
		os.Exit(1)
	}

	if len(result.Data) == 0 {
		fmt.Println("No currencies found.")
		return
	}

	fmt.Printf("Currencies (%d total):\n", result.Total)
	for _, c := range result.Data {
		fmt.Printf("  %s | %s | %s\n", c.UUID, c.Code, c.Name)
	}
}

func runCurrencyGet(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	idOrCode := args[0]

	var currency *db.Currency
	var err error

	if id, parseErr := uuid.Parse(idOrCode); parseErr == nil {
		currency, err = currencyService.GetByID(ctx, id)
	} else {
		currency, err = currencyService.GetByCode(ctx, idOrCode)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting currency: %v\n", err)
		os.Exit(1)
	}

	if currency == nil {
		fmt.Fprintf(os.Stderr, "Currency not found: %s\n", idOrCode)
		os.Exit(1)
	}

	fmt.Printf("Currency:\n")
	fmt.Printf("  UUID: %s\n", currency.UUID)
	fmt.Printf("  Code: %s\n", currency.Code)
	fmt.Printf("  Name: %s\n", currency.Name)
	if currency.Description != nil {
		fmt.Printf("  Description: %s\n", *currency.Description)
	}
	fmt.Printf("  Created: %s\n", currency.CreatedAt)
}
