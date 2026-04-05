package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wispberry-tech/kero-exchange/internal/db"
)

type DashboardService struct {
	pool *pgxpool.Pool
}

func NewDashboardService(pool *pgxpool.Pool) *DashboardService {
	return &DashboardService{pool: pool}
}

type DashboardSummary struct {
	WalletCount int
	Balances    []db.BalanceSummary
}

func (s *DashboardService) GetSummary(ctx context.Context, walletID uuid.UUID) (*DashboardSummary, error) {
	var walletCount int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet`).Scan(&walletCount)
	if err != nil {
		return nil, fmt.Errorf("get wallet count: %w", err)
	}

	balances, err := db.GetBalanceSummaryByWallet(ctx, s.pool, walletID)
	if err != nil {
		return nil, fmt.Errorf("get balance summary: %w", err)
	}

	return &DashboardSummary{
		WalletCount: walletCount,
		Balances:    balances,
	}, nil
}
