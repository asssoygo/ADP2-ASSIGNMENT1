package usecase

import (
	"context"

	"notification-service/internal/domain"
)

type EventProcessor interface {
	ProcessPaymentEvent(event domain.PaymentEvent) error
}

type EmailSender interface {
	Send(ctx context.Context, event domain.PaymentEvent) error
}
