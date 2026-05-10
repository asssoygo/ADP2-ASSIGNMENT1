package usecase

import (
	"context"
	"time"

	"order-service/internal/domain"
)

type OrderRepository interface {
	Create(order *domain.Order) error
	GetByID(id string) (*domain.Order, error)
	UpdateStatus(id string, status string) error
	GetByAmountRange(minAmount, maxAmount int64) ([]domain.Order, error)
}

type CacheRepository interface {
	Get(ctx context.Context, key string) (*domain.Order, error)
	Set(ctx context.Context, key string, order *domain.Order, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type PaymentClient interface {
	CreatePayment(orderID string, amount int64) (string, error)
}
