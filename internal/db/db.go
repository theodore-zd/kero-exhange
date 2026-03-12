package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaginationParams struct {
	Page     int
	PageSize int
}

func (p PaginationParams) Normalize() PaginationParams {
	page := p.Page
	if page < 1 {
		page = 1
	}
	pageSize := p.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return PaginationParams{Page: page, PageSize: pageSize}
}

func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

type PaginatedResult[T any] struct {
	Data       []T
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, connString)
}

func Paginate[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	baseQuery string,
	countQuery string,
	args []any,
	pageSize int,
	offset int,
	scanFunc func(rows pgx.Rows) ([]T, error),
) (PaginatedResult[T], error) {
	var total int64
	err := pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return PaginatedResult[T]{}, fmt.Errorf("count query failed: %w", err)
	}

	paginatedQuery := fmt.Sprintf("%s LIMIT $%d OFFSET $%d", baseQuery, len(args)+1, len(args)+2)
	args = append(args, pageSize, offset)

	rows, err := pool.Query(ctx, paginatedQuery, args...)
	if err != nil {
		return PaginatedResult[T]{}, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	data, err := scanFunc(rows)
	if err != nil {
		return PaginatedResult[T]{}, fmt.Errorf("scan failed: %w", err)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return PaginatedResult[T]{
		Data:       data,
		Total:      total,
		Page:       offset/pageSize + 1,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}
