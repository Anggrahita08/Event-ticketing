package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"project-ticketing-golang/internal/model"
	"project-ticketing-golang/internal/repository"
)

type EventService struct {
	eventRepo repository.EventRepository
}

func NewEventService(eventRepo repository.EventRepository) *EventService {
	return &EventService{eventRepo: eventRepo}
}

func (s *EventService) CreateEvent(ctx context.Context, adminID string, req model.CreateEventRequest) (*model.Event, error) {
	startAt, _ := time.Parse(time.RFC3339, req.StartAt)
	endAt, _ := time.Parse(time.RFC3339, req.EndAt)

	if startAt.Before(time.Now()) {
		return nil, errors.New("start_at harus di masa depan")
	}
	if !endAt.After(startAt) {
		return nil, errors.New("end_at harus setelah start_at")
	}

	event := &model.Event{
		Name:        req.Name,
		Description: req.Description,
		Venue:       req.Venue,
		StartAt:     startAt,
		EndAt:       endAt,
		Price:       req.Price,
		Quota:       req.Quota,
		Available:   req.Quota,
		CreatedBy:   adminID,
	}

	tx, err := s.eventRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	err = s.eventRepo.Create(ctx, tx, event)
	if err != nil {
		return nil, err
	}

	afterJSON, _ := json.Marshal(event)
	audit := &model.AuditLog{
		ActorID:    &adminID,
		ActorType:  "user",
		EntityType: "event",
		EntityID:   event.ID,
		Action:     "create",
		Before:     nil,
		After:      afterJSON,
	}
	s.eventRepo.InsertAuditLog(ctx, tx, audit)

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *EventService) UpdateEvent(ctx context.Context, adminID, id string, req model.UpdateEventRequest) (*model.Event, error) {
	tx, err := s.eventRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	event, err := s.eventRepo.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	if event.Status == "cancelled" {
		return nil, errors.New("event cancelled tidak bisa diubah")
	}

	if event.Version != req.Version {
		return nil, errors.New("VERSION_CONFLICT")
	}

	beforeJSON, _ := json.Marshal(event)

	startAt, _ := time.Parse(time.RFC3339, req.StartAt)
	endAt, _ := time.Parse(time.RFC3339, req.EndAt)

	terpakai := event.Quota - event.Available
	if req.Quota < terpakai {
		return nil, errors.New("quota tidak boleh lebih kecil dari tiket terpakai")
	}

	event.Name = req.Name
	event.Description = req.Description
	event.Venue = req.Venue
	event.StartAt = startAt
	event.EndAt = endAt
	event.Price = req.Price
	event.Quota = req.Quota
	event.Available = req.Quota - terpakai

	err = s.eventRepo.Update(ctx, tx, event)
	if err != nil {
		return nil, err
	}

	event.Version++
	event.UpdatedAt = time.Now()

	afterJSON, _ := json.Marshal(event)
	audit := &model.AuditLog{
		ActorID:    &adminID,
		ActorType:  "user",
		EntityType: "event",
		EntityID:   event.ID,
		Action:     "update",
		Before:     beforeJSON,
		After:      afterJSON,
	}
	s.eventRepo.InsertAuditLog(ctx, tx, audit)

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *EventService) PublishEvent(ctx context.Context, adminID, id string) (*model.Event, error) {
	tx, err := s.eventRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	event, err := s.eventRepo.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		return nil, model.ErrNotFound
	}

	if event.Status != "draft" {
		return nil, errors.New("INVALID_STATE")
	}
	if event.StartAt.Before(time.Now()) {
		return nil, errors.New("start_at sudah lewat")
	}

	beforeJSON, _ := json.Marshal(event)
	s.eventRepo.UpdateStatus(ctx, tx, id, "published")
	event.Status = "published"
	event.Version++
	afterJSON, _ := json.Marshal(event)

	audit := &model.AuditLog{
		ActorID: &adminID, ActorType: "user", EntityType: "event",
		EntityID: event.ID, Action: "publish", Before: beforeJSON, After: afterJSON,
	}
	s.eventRepo.InsertAuditLog(ctx, tx, audit)

	tx.Commit()
	return event, nil
}

func (s *EventService) CancelEvent(ctx context.Context, adminID, id string, req model.CancelEventRequest) (*model.Event, int64, error) {
	if len(req.Reason) < 5 || len(req.Reason) > 255 {
		return nil, 0, errors.New("reason wajib, 5-255 karakter")
	}

	tx, err := s.eventRepo.BeginTx(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer tx.Rollback()

	event, err := s.eventRepo.FindByIDForUpdate(ctx, tx, id)
	if err != nil {
		return nil, 0, model.ErrNotFound
	}

	if event.Status == "cancelled" {
		return nil, 0, errors.New("INVALID_STATE")
	}

	beforeJSON, _ := json.Marshal(event)

	// Ubah status event
	s.eventRepo.UpdateStatus(ctx, tx, id, "cancelled")
	event.Status = "cancelled"
	event.Version++

	// Batalkan semua order pending & paid
	cancelledOrders, err := s.eventRepo.CancelPendingPaidOrders(ctx, tx, id)
	if err != nil {
		return nil, 0, err
	}

	afterJSON, _ := json.Marshal(event)
	audit := &model.AuditLog{
		ActorID: &adminID, ActorType: "user", EntityType: "event",
		EntityID: event.ID, Action: "cancel", Before: beforeJSON, After: afterJSON,
	}
	s.eventRepo.InsertAuditLog(ctx, tx, audit)

	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}
	return event, cancelledOrders, nil
}
