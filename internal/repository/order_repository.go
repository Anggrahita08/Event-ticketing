package repository

import (
	"context"
	"database/sql"
	"fmt"
	"project-ticketing-golang/internal/model"
)

type OrderRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	GetEventForUpdate(ctx context.Context, tx *sql.Tx, eventID string) (*model.Event, error)
	CheckUserTicketLimit(ctx context.Context, tx *sql.Tx, userID, eventID string) (int, error)
	CreateOrder(ctx context.Context, tx *sql.Tx, order *model.Order) error
	ReduceEventQuota(ctx context.Context, tx *sql.Tx, eventID string, quantity int) error
	FindOrderByID(ctx context.Context, id string) (*model.Order, error)
	FindOrderByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.Order, error)
	UpdateOrderStatus(ctx context.Context, tx *sql.Tx, id string, status string) error
	ReturnEventQuota(ctx context.Context, tx *sql.Tx, eventID string, quantity int) error
	GetIdempotencyKey(ctx context.Context, tx *sql.Tx, userID, key string) (*model.IdempotencyKey, error)
	SaveIdempotencyKey(ctx context.Context, tx *sql.Tx, ik *model.IdempotencyKey) error

	// BARU: Untuk Get List Orders
	GetOrders(ctx context.Context, userID string, isAdmin bool, status, eventID string, page, limit int) ([]*model.Order, int, error)
}

type orderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *orderRepository) GetEventForUpdate(ctx context.Context, tx *sql.Tx, eventID string) (*model.Event, error) {
	query := `SELECT id, name, start_at, end_at, price, quota, available, status FROM events WHERE id = $1 FOR UPDATE`
	row := tx.QueryRowContext(ctx, query, eventID)
	var e model.Event
	err := row.Scan(&e.ID, &e.Name, &e.StartAt, &e.EndAt, &e.Price, &e.Quota, &e.Available, &e.Status)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *orderRepository) CheckUserTicketLimit(ctx context.Context, tx *sql.Tx, userID, eventID string) (int, error) {
	query := `SELECT COALESCE(SUM(quantity), 0) FROM orders WHERE user_id = $1 AND event_id = $2 AND status IN ('pending', 'paid')`
	var total int
	err := tx.QueryRowContext(ctx, query, userID, eventID).Scan(&total)
	return total, err
}

func (r *orderRepository) CreateOrder(ctx context.Context, tx *sql.Tx, order *model.Order) error {
	query := `INSERT INTO orders (user_id, event_id, quantity, unit_price, total_price, status, expires_at) 
	          VALUES ($1, $2, $3, $4, $5, 'pending', $6) 
	          RETURNING id, created_at, updated_at`
	return tx.QueryRowContext(ctx, query, order.UserID, order.EventID, order.Quantity, order.UnitPrice, order.TotalPrice, order.ExpiresAt).
		Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
}

func (r *orderRepository) ReduceEventQuota(ctx context.Context, tx *sql.Tx, eventID string, quantity int) error {
	query := `UPDATE events SET available = available - $1, updated_at = NOW() WHERE id = $2 AND available >= $1`
	res, err := tx.ExecContext(ctx, query, quantity, eventID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *orderRepository) FindOrderByID(ctx context.Context, id string) (*model.Order, error) {
	query := `SELECT o.id, o.user_id, o.event_id, e.name, e.start_at, o.quantity, o.unit_price, o.total_price, o.status, o.expires_at, o.paid_at, o.cancelled_at, o.created_at, o.updated_at 
	          FROM orders o JOIN events e ON o.event_id = e.id WHERE o.id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	var o model.Order
	var evName string
	var evStartAt sql.NullTime
	err := row.Scan(&o.ID, &o.UserID, &o.EventID, &evName, &evStartAt, &o.Quantity, &o.UnitPrice, &o.TotalPrice, &o.Status, &o.ExpiresAt, &o.PaidAt, &o.CancelledAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if evStartAt.Valid {
		o.Event = &model.EventInfo{ID: o.EventID, Name: evName, StartAt: evStartAt.Time}
	}
	return &o, nil
}

func (r *orderRepository) FindOrderByIDForUpdate(ctx context.Context, tx *sql.Tx, id string) (*model.Order, error) {
	query := `SELECT id, user_id, event_id, quantity, status, expires_at FROM orders WHERE id = $1 FOR UPDATE`
	row := tx.QueryRowContext(ctx, query, id)
	var o model.Order
	err := row.Scan(&o.ID, &o.UserID, &o.EventID, &o.Quantity, &o.Status, &o.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *orderRepository) UpdateOrderStatus(ctx context.Context, tx *sql.Tx, id string, status string) error {
	query := `UPDATE orders SET status = $1, updated_at = NOW()`
	args := []interface{}{status}

	if status == "paid" {
		query += `, paid_at = NOW()`
	} else if status == "cancelled" {
		query += `, cancelled_at = NOW()`
	}

	query += ` WHERE id = $2`
	args = append(args, id)

	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

func (r *orderRepository) ReturnEventQuota(ctx context.Context, tx *sql.Tx, eventID string, quantity int) error {
	query := `UPDATE events SET available = available + $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.ExecContext(ctx, query, quantity, eventID)
	return err
}

func (r *orderRepository) GetIdempotencyKey(ctx context.Context, tx *sql.Tx, userID, key string) (*model.IdempotencyKey, error) {
	query := `SELECT key, user_id, request_hash, response_status, response_body FROM idempotency_keys WHERE user_id = $1 AND key = $2 FOR UPDATE`
	row := tx.QueryRowContext(ctx, query, userID, key)
	var ik model.IdempotencyKey
	err := row.Scan(&ik.Key, &ik.UserID, &ik.RequestHash, &ik.ResponseStatus, &ik.ResponseBody)
	if err != nil {
		return nil, err
	}
	return &ik, nil
}

func (r *orderRepository) SaveIdempotencyKey(ctx context.Context, tx *sql.Tx, ik *model.IdempotencyKey) error {
	query := `INSERT INTO idempotency_keys (key, user_id, request_hash, response_status, response_body) VALUES ($1, $2, $3, $4, $5)`
	_, err := tx.ExecContext(ctx, query, ik.Key, ik.UserID, ik.RequestHash, ik.ResponseStatus, ik.ResponseBody)
	return err
}

// FUNGSI BARU: GetOrders untuk List
func (r *orderRepository) GetOrders(ctx context.Context, userID string, isAdmin bool, status, eventID string, page, limit int) ([]*model.Order, int, error) {
	baseQuery := `FROM orders o JOIN events e ON o.event_id = e.id WHERE 1=1`
	var args []interface{}
	argId := 1

	if !isAdmin {
		baseQuery += fmt.Sprintf(" AND o.user_id = $%d", argId)
		args = append(args, userID)
		argId++
	}
	if status != "" {
		baseQuery += fmt.Sprintf(" AND o.status = $%d", argId)
		args = append(args, status)
		argId++
	}
	if eventID != "" {
		baseQuery += fmt.Sprintf(" AND o.event_id = $%d", argId)
		args = append(args, eventID)
		argId++
	}

	var totalItems int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := fmt.Sprintf(`SELECT o.id, o.user_id, o.event_id, e.name, e.start_at, o.quantity, o.unit_price, o.total_price, o.status, o.expires_at, o.paid_at, o.cancelled_at, o.created_at, o.updated_at %s ORDER BY o.created_at DESC LIMIT $%d OFFSET $%d`, baseQuery, argId, argId+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*model.Order
	for rows.Next() {
		var o model.Order
		var evName string
		var evStartAt sql.NullTime
		err := rows.Scan(&o.ID, &o.UserID, &o.EventID, &evName, &evStartAt, &o.Quantity, &o.UnitPrice, &o.TotalPrice, &o.Status, &o.ExpiresAt, &o.PaidAt, &o.CancelledAt, &o.CreatedAt, &o.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		o.Event = &model.EventInfo{ID: o.EventID, Name: evName, StartAt: evStartAt.Time}
		orders = append(orders, &o)
	}
	return orders, totalItems, nil
}
