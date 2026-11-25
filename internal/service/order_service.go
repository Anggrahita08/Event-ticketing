package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"project-ticketing-golang/internal/model"
	"project-ticketing-golang/internal/repository"
)

type OrderService struct {
	orderRepo repository.OrderRepository
}

func NewOrderService(orderRepo repository.OrderRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID, idempotencyKey string, req model.CreateOrderRequest) (*model.Order, bool, error) {
	if req.Quantity < 1 || req.Quantity > 5 {
		return nil, false, errors.New("quantity harus antara 1-5")
	}

	bodyBytes, _ := json.Marshal(req)
	hash := sha256.Sum256(bodyBytes)
	reqHash := hex.EncodeToString(hash[:])

	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	existingKey, err := s.orderRepo.GetIdempotencyKey(ctx, tx, userID, idempotencyKey)
	if err == nil {
		if existingKey.RequestHash != reqHash {
			return nil, false, errors.New("IDEMPOTENCY_KEY_REUSED")
		}
		var prevOrder model.Order
		json.Unmarshal(existingKey.ResponseBody, &prevOrder)
		return &prevOrder, true, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}

	event, err := s.orderRepo.GetEventForUpdate(ctx, tx, req.EventID)
	if err != nil {
		return nil, false, model.ErrNotFound
	}

	if event.Status != "published" {
		return nil, false, errors.New("INVALID_STATE")
	}
	if event.StartAt.Before(time.Now()) {
		return nil, false, errors.New("INVALID_STATE")
	}

	existingTickets, err := s.orderRepo.CheckUserTicketLimit(ctx, tx, userID, req.EventID)
	if err != nil {
		return nil, false, err
	}
	if existingTickets+req.Quantity > 5 {
		return nil, false, errors.New("TICKET_LIMIT_EXCEEDED")
	}

	err = s.orderRepo.ReduceEventQuota(ctx, tx, req.EventID, req.Quantity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, errors.New("INSUFFICIENT_QUOTA")
		}
		return nil, false, err
	}

	order := &model.Order{
		UserID:     userID,
		EventID:    req.EventID,
		Quantity:   req.Quantity,
		UnitPrice:  event.Price,
		TotalPrice: float64(req.Quantity) * event.Price,
		Status:     "pending",
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}

	err = s.orderRepo.CreateOrder(ctx, tx, order)
	if err != nil {
		return nil, false, err
	}

	order.Event = &model.EventInfo{ID: event.ID, Name: event.Name, StartAt: event.StartAt}

	responseBody, _ := json.Marshal(order)
	ik := &model.IdempotencyKey{
		Key:            idempotencyKey,
		UserID:         userID,
		RequestHash:    reqHash,
		ResponseStatus: 201,
		ResponseBody:   responseBody,
	}
	err = s.orderRepo.SaveIdempotencyKey(ctx, tx, ik)
	if err != nil {
		return nil, false, err
	}

	if err := tx.Commit(); err != nil {
		return nil, false, err
	}

	return order, false, nil
}

func (s *OrderService) PayOrder(ctx context.Context, userID, orderID string) (*model.Order, error) {
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	order, err := s.orderRepo.FindOrderByIDForUpdate(ctx, tx, orderID)
	if err != nil {
		return nil, model.ErrNotFound
	}

	if order.UserID != userID {
		return nil, model.ErrNotFound
	}

	if order.Status != "pending" {
		return nil, errors.New("INVALID_STATE")
	}
	if order.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("INVALID_STATE")
	}

	s.orderRepo.UpdateOrderStatus(ctx, tx, orderID, "paid")
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.orderRepo.FindOrderByID(ctx, orderID)
}

func (s *OrderService) CancelOrder(ctx context.Context, userID, orderID string) (*model.Order, error) {
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	order, err := s.orderRepo.FindOrderByIDForUpdate(ctx, tx, orderID)
	if err != nil {
		return nil, model.ErrNotFound
	}

	if order.UserID != userID {
		return nil, model.ErrNotFound
	}

	if order.Status != "pending" && order.Status != "paid" {
		return nil, errors.New("INVALID_STATE")
	}

	event, err := s.orderRepo.GetEventForUpdate(ctx, tx, order.EventID)
	if err != nil {
		return nil, err
	}
	if event.StartAt.Before(time.Now()) {
		return nil, errors.New("INVALID_STATE")
	}

	s.orderRepo.UpdateOrderStatus(ctx, tx, orderID, "cancelled")
	s.orderRepo.ReturnEventQuota(ctx, tx, order.EventID, order.Quantity)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.orderRepo.FindOrderByID(ctx, orderID)
}

func (s *OrderService) GetOrders(ctx context.Context, userID string, isAdmin bool, status, eventID string, page, limit int) ([]*model.Order, int, error) {
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 10
	}
	if page < 1 {
		page = 1
	}
	return s.orderRepo.GetOrders(ctx, userID, isAdmin, status, eventID, page, limit)
}

func (s *OrderService) GetOrderByID(ctx context.Context, userID string, isAdmin bool, orderID string) (*model.Order, error) {
	order, err := s.orderRepo.FindOrderByID(ctx, orderID)
	if err != nil {
		return nil, model.ErrNotFound
	}
	if !isAdmin && order.UserID != userID {
		return nil, model.ErrNotFound
	}
	return order, nil
}
