package usecase

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"notification-service/internal/domain"
)

const maxNotifications = 50

type ProcessedNotification struct {
	MessageID     string    `json:"message_id"`
	OrderID       string    `json:"order_id"`
	CustomerEmail string    `json:"customer_email"`
	Amount        int64     `json:"amount"`
	ProcessedAt   time.Time `json:"processed_at"`
}

type NotificationUsecase struct {
	mu            sync.Mutex
	processed     map[string]bool
	notifications []ProcessedNotification
}

func NewNotificationUsecase() *NotificationUsecase {
	return &NotificationUsecase{
		processed:     make(map[string]bool),
		notifications: make([]ProcessedNotification, 0, maxNotifications),
	}
}

func (u *NotificationUsecase) ProcessPaymentEvent(event domain.PaymentEvent) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.processed[event.MessageID] {
		log.Printf("[Notification] Duplicate message skipped: %s", event.MessageID)
		return nil
	}

	// Demo: orders with "fail@" in the email simulate a processing failure so
	// the DLQ retry logic can be observed.
	if strings.Contains(event.CustomerEmail, "fail@") {
		return errors.New("simulated processing failure (fail@ address)")
	}

	log.Printf(
		"[Notification] Sent email to %s for Order #%s. Amount: $%d",
		event.CustomerEmail,
		event.OrderID,
		event.Amount,
	)

	u.processed[event.MessageID] = true
	u.notifications = append(u.notifications, ProcessedNotification{
		MessageID:     event.MessageID,
		OrderID:       event.OrderID,
		CustomerEmail: event.CustomerEmail,
		Amount:        event.Amount,
		ProcessedAt:   time.Now().UTC(),
	})
	if len(u.notifications) > maxNotifications {
		u.notifications = u.notifications[len(u.notifications)-maxNotifications:]
	}

	return nil
}

// GetRecentNotifications returns up to 50 processed events, newest first.
func (u *NotificationUsecase) GetRecentNotifications() []ProcessedNotification {
	u.mu.Lock()
	defer u.mu.Unlock()

	result := make([]ProcessedNotification, len(u.notifications))
	copy(result, u.notifications)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}
