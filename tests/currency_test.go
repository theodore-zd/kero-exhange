package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestCurrencyList_Empty(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM currencies")

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/currencies")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) != 0 {
		t.Errorf("Expected empty data array, got %d items", len(data))
	}
}

func TestCurrencyList_WithData(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM currencies")

	currency, err := createTestCurrency(ctx, "USD", "US Dollar")
	if err != nil {
		t.Fatalf("Failed to create test currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/currencies")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	data, _ := parseListResponse(t, body)

	if len(data) == 0 {
		t.Error("Expected at least one currency in data array")
	}
}

func TestCurrencyList_Pagination(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM currencies")

	for i := 0; i < 25; i++ {
		currency, err := createTestCurrency(ctx, fmt.Sprintf("CUR%02d", i), fmt.Sprintf("Currency %d", i))
		if err != nil {
			t.Fatalf("Failed to create currency: %v", err)
		}
		defer deleteTestCurrency(ctx, currency.UUID)
	}

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/currencies?page=1&page_size=10")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	_, meta := parseListResponse(t, body)

	if meta["page"] != float64(1) {
		t.Errorf("Expected page 1, got %v", meta["page"])
	}
	if meta["page_size"] != float64(10) {
		t.Errorf("Expected page_size 10, got %v", meta["page_size"])
	}
}

func TestCurrencyGet_Success(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM currencies")

	currency, err := createTestCurrency(ctx, "EUR", "Euro")
	if err != nil {
		t.Fatalf("Failed to create test currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/currencies/" + currency.UUID.String())
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data in response, got: %s", string(body))
	}

	if data["uuid"] != currency.UUID.String() {
		t.Errorf("Expected uuid %s, got %s", currency.UUID, data["uuid"])
	}
	if data["code"] != "EUR" {
		t.Errorf("Expected code EUR, got %s", data["code"])
	}
}

func TestCurrencyGet_NotFound(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	fakeID := uuid.New().String()
	resp, err := http.Get(server.URL + "/api/v1/currencies/" + fakeID)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestCurrencyGet_InvalidUUID(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/currencies/invalid-uuid")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestCurrencyGetByCode_Success(t *testing.T) {
	ctx := context.Background()
	testPool.Exec(ctx, "DELETE FROM currencies")

	currency, err := createTestCurrency(ctx, "GBP", "British Pound")
	if err != nil {
		t.Fatalf("Failed to create test currency: %v", err)
	}
	defer deleteTestCurrency(ctx, currency.UUID)

	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/currencies/code/GBP")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected data in response, got: %s", string(body))
	}

	if data["code"] != "GBP" {
		t.Errorf("Expected code GBP, got %s", data["code"])
	}
}

func TestCurrencyGetByCode_NotFound(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/currencies/code/NOTEXIST")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestCurrencyGetByCode_EmptyCode(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/currencies/code/")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}
