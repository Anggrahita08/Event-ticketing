package repository

import (
	"context"
	"database/sql"
	"fmt"
	"project-ticketing-golang/internal/model"
)

type AuditLogRepository interface {
	GetLogs(ctx context.Context, entityType, entityID string, page, limit int) ([]*model.AuditLogResponse, int, error)
}

type auditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) GetLogs(ctx context.Context, entityType, entityID string, page, limit int) ([]*model.AuditLogResponse, int, error) {
	baseQuery := `FROM audit_logs WHERE 1=1`
	var args []interface{}
	argId := 1

	if entityType != "" {
		baseQuery += fmt.Sprintf(" AND entity_type = $%d", argId)
		args = append(args, entityType)
		argId++
	}
	if entityID != "" {
		baseQuery += fmt.Sprintf(" AND entity_id = $%d", argId)
		args = append(args, entityID)
		argId++
	}

	var totalItems int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := fmt.Sprintf(`SELECT id, actor_id, actor_type, entity_type, entity_id, action, before, after, created_at %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, baseQuery, argId, argId+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*model.AuditLogResponse
	for rows.Next() {
		var l model.AuditLogResponse
		err := rows.Scan(&l.ID, &l.ActorID, &l.ActorType, &l.EntityType, &l.EntityID, &l.Action, &l.Before, &l.After, &l.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	return logs, totalItems, nil
}
