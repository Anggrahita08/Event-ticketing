package repository

import (
	"context"
	"database/sql"
	"project-ticketing-golang/internal/model"
)

type EventRepository interface {
	Create(ctx context.Context, tx *sql.Tx, event *model.Event) error
	FindByID(ctx context.Context, id string) (*model.Event, error)
	FindByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.Event, error)
	Update(ctx context.Context, tx *sql.Tx, event *model.Event) error
	UpdateStatus(ctx context.Context, tx *sql.Tx, id string, status string) error
	InsertAuditLog(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
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
	query := `SELECT id, name, description, venue, start_at, end_at, price, quota, available, status, version, created_by, created_at, updated_at FROM events WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	var e model.Event
	var desc sql.NullString
	err := row.Scan(&e.ID, &e.Name, &desc, &e.Venue, &e.StartAt, &e.EndAt, &e.Price, &e.Quota, &e.Available, &e.Status, &e.Version, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.Description = desc.String
	return &e, nil
}

func (r *eventRepository) FindByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.Event, error) {
	query := `SELECT id, name, description, venue, start_at, end_at, price, quota, available, status, version, created_by, created_at, updated_at FROM events WHERE id = $1 FOR UPDATE`
	row := tx.QueryRowContext(ctx, query, id)
	var e model.Event
	var desc sql.NullString
	err := row.Scan(&e.ID, &e.Name, &desc, &e.Venue, &e.StartAt, &e.EndAt, &e.Price, &e.Quota, &e.Available, &e.Status, &e.Version, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	e.Description = desc.String
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

func (r *eventRepository) InsertAuditLog(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error {
	query := `INSERT INTO audit_logs (actor_id, actor_type, entity_type, entity_id, action, before, after) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := tx.ExecContext(ctx, query, log.ActorID, log.ActorType, log.EntityType, log.EntityID, log.Action, log.Before, log.After)
	return err
}
