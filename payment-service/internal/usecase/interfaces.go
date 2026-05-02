package usecase

import "payment-service/internal/domain"

type PaymentCompletedEvent struct {
	MessageID     string `json:"message_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type EventPublisher interface {
	PublishPaymentCompleted(event PaymentCompletedEvent) error
}
type PaymentRepository interface {
	Create(payment *domain.Payment) error
	GetByOrderID(orderID string) (*domain.Payment, error)
	ListByStatus(status string) ([]domain.Payment, error)
}
