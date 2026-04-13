package usecase

import "order-service/internal/domain"

type OrderRepository interface {
	Create(order *domain.Order) error
	GetByID(id string) (*domain.Order, error)
	UpdateStatus(id string, status string) error
	GetByAmountRange(minAmount, maxAmount int64) ([]domain.Order, error)
}

type PaymentClient interface {
	CreatePayment(orderID string, amount int64) (string, error)
}
