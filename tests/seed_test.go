package tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/wispberry-tech/kero-exchange/internal/db"
	"github.com/wispberry-tech/kero-exchange/internal/services"
)

var (
	testSeedWalletAUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testSeedWalletBUUID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testSeedProviderUUID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

func cleanupSeedData(ctx context.Context) {
	testPool.Exec(ctx, "DELETE FROM access_tokens WHERE wallet_id IN ($1, $2)", testSeedWalletAUUID, testSeedWalletBUUID)
	testPool.Exec(ctx, "DELETE FROM transactions WHERE wallet_id IN ($1, $2)", testSeedWalletAUUID, testSeedWalletBUUID)
	testPool.Exec(ctx, "DELETE FROM balances WHERE wallet_id IN ($1, $2)", testSeedWalletAUUID, testSeedWalletBUUID)
	testPool.Exec(ctx, "DELETE FROM wallet WHERE uuid IN ($1, $2)", testSeedWalletAUUID, testSeedWalletBUUID)
	testPool.Exec(ctx, "DELETE FROM providers WHERE uuid = $1", testSeedProviderUUID)
}

func TestSeedMode(t *testing.T) {
	ctx := context.Background()
	cleanupSeedData(ctx)
	defer cleanupSeedData(ctx)

	seedSvc := services.NewSeedService(testPool)

	// First seed call
	if err := seedSvc.Seed(ctx); err != nil {
		t.Fatalf("Seed() returned error: %v", err)
	}

	// Verify wallet A exists
	walletA, err := db.GetWalletByUUID(ctx, testPool, testSeedWalletAUUID)
	if err != nil {
		t.Fatalf("Failed to get wallet A: %v", err)
	}
	if walletA == nil {
		t.Fatal("Expected wallet A to exist after seeding")
	}
	if walletA.UUID != testSeedWalletAUUID {
		t.Errorf("Expected wallet A UUID %s, got %s", testSeedWalletAUUID, walletA.UUID)
	}

	// Verify wallet B exists
	walletB, err := db.GetWalletByUUID(ctx, testPool, testSeedWalletBUUID)
	if err != nil {
		t.Fatalf("Failed to get wallet B: %v", err)
	}
	if walletB == nil {
		t.Fatal("Expected wallet B to exist after seeding")
	}
	if walletB.UUID != testSeedWalletBUUID {
		t.Errorf("Expected wallet B UUID %s, got %s", testSeedWalletBUUID, walletB.UUID)
	}

	// Verify provider exists
	var providerCount int
	err = testPool.QueryRow(ctx, "SELECT COUNT(*) FROM providers WHERE uuid = $1", testSeedProviderUUID).Scan(&providerCount)
	if err != nil {
		t.Fatalf("Failed to query provider: %v", err)
	}
	if providerCount != 1 {
		t.Errorf("Expected 1 seed provider, got %d", providerCount)
	}

	// Idempotency: second call should not error
	if err := seedSvc.Seed(ctx); err != nil {
		t.Fatalf("Second Seed() call returned error: %v", err)
	}

	// Verify still only one wallet A (no duplicates)
	var walletACount int
	err = testPool.QueryRow(ctx, "SELECT COUNT(*) FROM wallet WHERE uuid = $1", testSeedWalletAUUID).Scan(&walletACount)
	if err != nil {
		t.Fatalf("Failed to count wallet A: %v", err)
	}
	if walletACount != 1 {
		t.Errorf("Expected exactly 1 wallet A after idempotent seed, got %d", walletACount)
	}
}
