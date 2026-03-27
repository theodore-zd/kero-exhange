package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateAuditLogParams struct {
	Action     string
	EntityType string
	EntityID   uuid.UUID
	Details    map[string]interface{}
	AdminUser  string
	IPAddress  string
	UserAgent  string
}

type AuditLog struct {
	UUID       uuid.UUID
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	Details    map[string]interface{}
	AdminUser  string
	IPAddress  string
	UserAgent  string
	CreatedAt  time.Time
}

type AuditLogFilter struct {
	Action     string
	EntityType string
	EntityID   uuid.UUID
	StartDate  *time.Time
	EndDate    *time.Time
}

func CreateAuditLog(ctx context.Context, pool *pgxpool.Pool, params CreateAuditLogParams) (*AuditLog, error) {
	var detailsJSON []byte
	if params.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(params.Details)
		if err != nil {
			return nil, fmt.Errorf("marshal audit log details: %w", err)
		}
	} else {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO admin_audit_logs (action, entity_type, entity_id, details, admin_user, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING uuid, action, entity_type, entity_id, details, admin_user, ip_address, user_agent, created_at
	`
	var al AuditLog
	var entityID *uuid.UUID
	if params.EntityID != uuid.Nil {
		entityID = &params.EntityID
	}

	err := pool.QueryRow(ctx, query,
		params.Action,
		params.EntityType,
		entityID,
		detailsJSON,
		params.AdminUser,
		params.IPAddress,
		params.UserAgent,
	).Scan(
		&al.UUID,
		&al.Action,
		&al.EntityType,
		&al.EntityID,
		&al.Details,
		&al.AdminUser,
		&al.IPAddress,
		&al.UserAgent,
		&al.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create audit log: %w", err)
	}
	return &al, nil
}

func GetAuditLogs(ctx context.Context, pool *pgxpool.Pool, params PaginationParams, filter AuditLogFilter) (PaginatedResult[AuditLog], error) {
	params = params.Normalize()

	baseQuery := `
		SELECT uuid, action, entity_type, entity_id, details, admin_user, ip_address, user_agent, created_at
		FROM admin_audit_logs
	`
	countQuery := `SELECT COUNT(*) FROM admin_audit_logs`
	args := []any{}
	argIdx := 1
	hasWhere := false

	if filter.Action != "" {
		whereBase := fmt.Sprintf(" WHERE action = $%d", argIdx)
		whereCount := fmt.Sprintf(" WHERE action = $%d", argIdx)
		baseQuery += whereBase
		countQuery += whereCount
		args = append(args, filter.Action)
		argIdx++
		hasWhere = true
	}

	if filter.EntityType != "" {
		connectorBase := " WHERE "
		connectorCount := " WHERE "
		if hasWhere {
			connectorBase = " AND "
			connectorCount = " AND "
		}
		whereBase := fmt.Sprintf("%sentity_type = $%d", connectorBase, argIdx)
		whereCount := fmt.Sprintf("%sentity_type = $%d", connectorCount, argIdx)
		baseQuery += whereBase
		countQuery += whereCount
		args = append(args, filter.EntityType)
		argIdx++
		hasWhere = true
	}

	if filter.EntityID != uuid.Nil {
		connectorBase := " WHERE "
		connectorCount := " WHERE "
		if hasWhere {
			connectorBase = " AND "
			connectorCount = " AND "
		}
		whereBase := fmt.Sprintf("%sentity_id = $%d", connectorBase, argIdx)
		whereCount := fmt.Sprintf("%sentity_id = $%d", connectorCount, argIdx)
		baseQuery += whereBase
		countQuery += whereCount
		args = append(args, filter.EntityID)
		argIdx++
		hasWhere = true
	}

	if filter.StartDate != nil {
		connectorBase := " WHERE "
		connectorCount := " WHERE "
		if hasWhere {
			connectorBase = " AND "
			connectorCount = " AND "
		}
		whereBase := fmt.Sprintf("%screated_at >= $%d", connectorBase, argIdx)
		whereCount := fmt.Sprintf("%screated_at >= $%d", connectorCount, argIdx)
		baseQuery += whereBase
		countQuery += whereCount
		args = append(args, filter.StartDate)
		argIdx++
		hasWhere = true
	}

	if filter.EndDate != nil {
		connectorBase := " WHERE "
		connectorCount := " WHERE "
		if hasWhere {
			connectorBase = " AND "
			connectorCount = " AND "
		}
		whereBase := fmt.Sprintf("%screated_at <= $%d", connectorBase, argIdx)
		whereCount := fmt.Sprintf("%screated_at <= $%d", connectorCount, argIdx)
		baseQuery += whereBase
		countQuery += whereCount
		args = append(args, filter.EndDate)
	}

	baseQuery += " ORDER BY created_at DESC"

	return Paginate(ctx, pool, baseQuery, countQuery, args, params.PageSize, params.Offset(),
		func(rows pgx.Rows) ([]AuditLog, error) {
			var logs []AuditLog
			for rows.Next() {
				var al AuditLog
				var detailsJSON []byte
				if err := rows.Scan(&al.UUID, &al.Action, &al.EntityType, &al.EntityID, &detailsJSON, &al.AdminUser, &al.IPAddress, &al.UserAgent, &al.CreatedAt); err != nil {
					return nil, err
				}
				if len(detailsJSON) > 0 {
					json.Unmarshal(detailsJSON, &al.Details)
				} else {
					al.Details = make(map[string]interface{})
				}
				logs = append(logs, al)
			}
			return logs, nil
		},
	)
}
