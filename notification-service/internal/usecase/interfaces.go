package usecase

import "notification-service/internal/domain"

type EventProcessor interface {
	ProcessPaymentEvent(event domain.PaymentEvent) error
}
