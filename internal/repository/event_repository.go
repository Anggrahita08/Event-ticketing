package repository

import (
	"context"
	"database/sql"
	"fmt"
	"project-ticketing-golang/internal/model"
	"strings"
)

type EventRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	Create(ctx context.Context, tx *sql.Tx, event *model.Event) error
	FindByID(ctx context.Context, id string) (*model.Event, error)
	FindByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.Event, error)
	Update(ctx context.Context, tx *sql.Tx, event *model.Event) error
	UpdateStatus(ctx context.Context, tx *sql.Tx, id string, status string) error
	CancelPendingPaidOrders(ctx context.Context, tx *sql.Tx, eventID string) (int64, error)
	InsertAuditLog(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error

	// Tambahan untuk Event Publik
	GetEvents(ctx context.Context, filter map[string]interface{}, page, limit int) ([]*model.Event, int, error)
}

type eventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *eventRepository) Create(ctx context.Context, tx *sql.Tx, event *model.Event) error {
	query := `INSERT INTO events (name, description, venue, start_at, end_at, price, quota, available, status, version, created_by) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $7, 'draft', 1, $8) 
	          RETURNING id, status, version, created_at, updated_at`
	return tx.QueryRowContext(ctx, query, event.Name, event.Description, event.Venue, event.StartAt, event.EndAt, event.Price, event.Quota, event.CreatedBy).
		Scan(&event.ID, &event.Status, &event.Version, &event.CreatedAt, &event.UpdatedAt)
}

func (r *eventRepository) FindByID(ctx context.Context, id string) (*model.Event, error) {
	query := `SELECT id, name, COALESCE(description, ''), venue, start_at, end_at, price, quota, available, status, version, created_by, created_at, updated_at FROM events WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	var e model.Event
	err := row.Scan(&e.ID, &e.Name, &e.Description, &e.Venue, &e.StartAt, &e.EndAt, &e.Price, &e.Quota, &e.Available, &e.Status, &e.Version, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *eventRepository) FindByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.Event, error) {
	query := `SELECT id, name, COALESCE(description, ''), venue, start_at, end_at, price, quota, available, status, version, created_by, created_at, updated_at FROM events WHERE id = $1 FOR UPDATE`
	row := tx.QueryRowContext(ctx, query, id)
	var e model.Event
	err := row.Scan(&e.ID, &e.Name, &e.Description, &e.Venue, &e.StartAt, &e.EndAt, &e.Price, &e.Quota, &e.Available, &e.Status, &e.Version, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *eventRepository) Update(ctx context.Context, tx *sql.Tx, event *model.Event) error {
	query := `UPDATE events SET name=$1, description=$2, venue=$3, start_at=$4, end_at=$5, price=$6, quota=$7, available=$8, version=version+1, updated_at=NOW() WHERE id=$9`
	_, err := tx.ExecContext(ctx, query, event.Name, event.Description, event.Venue, event.StartAt, event.EndAt, event.Price, event.Quota, event.Available, event.ID)
	return err
}

func (r *eventRepository) UpdateStatus(ctx context.Context, tx *sql.Tx, id string, status string) error {
	query := `UPDATE events SET status=$1, version=version+1, updated_at=NOW() WHERE id=$2`
	_, err := tx.ExecContext(ctx, query, status, id)
	return err
}

func (r *eventRepository) CancelPendingPaidOrders(ctx context.Context, tx *sql.Tx, eventID string) (int64, error) {
	query := `UPDATE orders SET status = 'cancelled', cancelled_at = NOW(), updated_at = NOW() WHERE event_id = $1 AND status IN ('pending', 'paid')`
	res, err := tx.ExecContext(ctx, query, eventID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *eventRepository) InsertAuditLog(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error {
	query := `INSERT INTO audit_logs (actor_id, actor_type, entity_type, entity_id, action, before, after) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := tx.ExecContext(ctx, query, log.ActorID, log.ActorType, log.EntityType, log.EntityID, log.Action, log.Before, log.After)
	return err
}

func (r *eventRepository) GetEvents(ctx context.Context, filter map[string]interface{}, page, limit int) ([]*model.Event, int, error) {
	baseQuery := `FROM events WHERE 1=1`
	var args []interface{}
	argId := 1

	if status, ok := filter["status"].(string); ok && status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argId)
		args = append(args, status)
		argId++
	}
	if search, ok := filter["search"].(string); ok && search != "" {
		baseQuery += fmt.Sprintf(" AND (LOWER(name) LIKE $%d OR LOWER(venue) LIKE $%d)", argId, argId)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argId++
	}
	if startFrom, ok := filter["start_from"].(string); ok && startFrom != "" {
		baseQuery += fmt.Sprintf(" AND start_at >= $%d", argId)
		args = append(args, startFrom)
		argId++
	}
	if startTo, ok := filter["start_to"].(string); ok && startTo != "" {
		baseQuery += fmt.Sprintf(" AND start_at <= $%d", argId)
		args = append(args, startTo)
		argId++
	}

	var totalItems int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	sortBy := "start_at"
	if s, ok := filter["sort"].(string); ok && s != "" {
		sortBy = s
	}
	order := "ASC"
	if o, ok := filter["order"].(string); ok && strings.ToLower(o) == "desc" {
		order = "DESC"
	}

	offset := (page - 1) * limit
	query := fmt.Sprintf(`SELECT id, name, COALESCE(description, ''), venue, start_at, end_at, price, quota, available, status, version, created_by, created_at, updated_at %s ORDER BY %s %s LIMIT $%d OFFSET $%d`, baseQuery, sortBy, order, argId, argId+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var e model.Event
		err := rows.Scan(&e.ID, &e.Name, &e.Description, &e.Venue, &e.StartAt, &e.EndAt, &e.Price, &e.Quota, &e.Available, &e.Status, &e.Version, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, &e)
	}
	return events, totalItems, nil
}
