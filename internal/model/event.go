package model

import "time"

type Event struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Venue       string    `json:"venue"`
	StartAt     time.Time `json:"start_at"`
	EndAt       time.Time `json:"end_at"`
	Price       float64   `json:"price"`
	Quota       int       `json:"quota"`
	Available   int       `json:"available"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	CreatedBy   string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateEventRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Venue       string  `json:"venue"`
	StartAt     string  `json:"start_at"` // RFC 3339
	EndAt       string  `json:"end_at"`   // RFC 3339
	Price       float64 `json:"price"`
	Quota       int     `json:"quota"`
}

type UpdateEventRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Venue       string  `json:"venue"`
	StartAt     string  `json:"start_at"`
	EndAt       string  `json:"end_at"`
	Price       float64 `json:"price"`
	Quota       int     `json:"quota"`
	Version     int     `json:"version"`
}

type CancelEventRequest struct {
	Reason string `json:"reason"`
}
