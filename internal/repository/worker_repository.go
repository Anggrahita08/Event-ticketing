package repository

import (
	"context"
	"database/sql"
	"project-ticketing-golang/internal/model"
)

type WorkerRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	GetExpiredOrdersForUpdate(ctx context.Context, tx *sql.Tx, limit int) ([]*model.Order, error)
	ExpireOrder(ctx context.Context, tx *sql.Tx, orderID string, eventID string, quantity int) error
	InsertAuditLogTx(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error
}

type workerRepository struct {
	db *sql.DB
}

func NewWorkerRepository(db *sql.DB) WorkerRepository {
	return &workerRepository{db: db}
}

func (r *workerRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *workerRepository) GetExpiredOrdersForUpdate(ctx context.Context, tx *sql.Tx, limit int) ([]*model.Order, error) {
	// FOR UPDATE SKIP LOCKED memastikan multi-instance safe
	query := `SELECT id, event_id, quantity, status, expires_at 
	          FROM orders 
	          WHERE status = 'pending' AND expires_at < NOW() 
	          LIMIT $1 
	          FOR UPDATE SKIP LOCKED`
	rows, err := tx.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var o model.Order
		err := rows.Scan(&o.ID, &o.EventID, &o.Quantity, &o.Status, &o.ExpiresAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

func (r *workerRepository) ExpireOrder(ctx context.Context, tx *sql.Tx, orderID string, eventID string, quantity int) error {
	// 1. Ubah status order menjadi expired
	queryOrder := `UPDATE orders SET status = 'expired', updated_at = NOW() WHERE id = $1 AND status = 'pending'`
	res, err := tx.ExecContext(ctx, queryOrder, orderID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil // Sudah di-expire oleh instance lain
	}

	// 2. Kembalikan kuota event
	queryEvent := `UPDATE events SET available = available + $1, updated_at = NOW() WHERE id = $2`
	_, err = tx.ExecContext(ctx, queryEvent, quantity, eventID)
	return err
}

func (r *workerRepository) InsertAuditLogTx(ctx context.Context, tx *sql.Tx, log *model.AuditLog) error {
	query := `INSERT INTO audit_logs (actor_id, actor_type, entity_type, entity_id, action, before, after) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := tx.ExecContext(ctx, query, log.ActorID, log.ActorType, log.EntityType, log.EntityID, log.Action, log.Before, log.After)
	return err
}
