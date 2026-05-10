package email

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"time"

	"notification-service/internal/domain"
)

type SimulatedEmailSender struct{}

func NewSimulatedEmailSender() *SimulatedEmailSender {
	return &SimulatedEmailSender{}
}

func (s *SimulatedEmailSender) Send(_ context.Context, event domain.PaymentEvent) error {
	time.Sleep(500 * time.Millisecond)

	if rand.Float32() < 0.3 {
		return errors.New("simulated email delivery failure")
	}

	log.Printf("[Email] Sent to %s for Order #%s. Amount: $%d",
		event.CustomerEmail, event.OrderID, event.Amount)
	return nil
}
