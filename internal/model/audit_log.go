package model

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ActorID    *string         `json:"actor_id"`
	ActorType  string          `json:"actor_type"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Action     string          `json:"action"`
	Before     json.RawMessage `json:"before"`
	After      json.RawMessage `json:"after"`
}

type AuditLogResponse struct {
	ID         string          `json:"id"`
	ActorID    *string         `json:"actor_id"`
	ActorType  string          `json:"actor_type"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	Action     string          `json:"action"`
	Before     json.RawMessage `json:"before"`
	After      json.RawMessage `json:"after"`
	CreatedAt  time.Time       `json:"created_at"`
}
