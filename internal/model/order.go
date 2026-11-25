package model

import "time"

type Order struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	EventID     string     `json:"event_id"`
	Event       *EventInfo `json:"event,omitempty"`
	Quantity    int        `json:"quantity"`
	UnitPrice   float64    `json:"unit_price"`
	TotalPrice  float64    `json:"total_price"`
	Status      string     `json:"status"`
	ExpiresAt   time.Time  `json:"expires_at"`
	PaidAt      *time.Time `json:"paid_at"`
	CancelledAt *time.Time `json:"cancelled_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type EventInfo struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	StartAt time.Time `json:"start_at"`
}

type CreateOrderRequest struct {
	EventID  string `json:"event_id"`
	Quantity int    `json:"quantity"`
}

// IdempotencyKey menyimpan hasil request agar request berulang
// dengan key yang sama tidak membuat order duplikat.
type IdempotencyKey struct {
	Key            string
	UserID         string
	RequestHash    string
	ResponseStatus int
	ResponseBody   []byte
}
