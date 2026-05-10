package usecase

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"notification-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

const (
	maxNotifications = 50
	maxAttempts      = 4 // 1 initial + 3 retries
	idempotencyTTL   = 24 * time.Hour
)

var retryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

type ProcessedNotification struct {
	MessageID     string    `json:"message_id"`
	OrderID       string    `json:"order_id"`
	CustomerEmail string    `json:"customer_email"`
	Amount        int64     `json:"amount"`
	ProcessedAt   time.Time `json:"processed_at"`
}

type NotificationUsecase struct {
	mu            sync.Mutex
	notifications []ProcessedNotification
	emailSender   EmailSender
	redisClient   *redis.Client
}

func NewNotificationUsecase(emailSender EmailSender, redisClient *redis.Client) *NotificationUsecase {
	return &NotificationUsecase{
		notifications: make([]ProcessedNotification, 0, maxNotifications),
		emailSender:   emailSender,
		redisClient:   redisClient,
	}
}

func (u *NotificationUsecase) ProcessPaymentEvent(event domain.PaymentEvent) error {
	ctx := context.Background()
	idempotencyKey := fmt.Sprintf("notification:%s", event.MessageID)

	exists, err := u.redisClient.Exists(ctx, idempotencyKey).Result()
	if err != nil {
		log.Printf("[Notification] Redis check failed for %s: %v", event.MessageID, err)
	} else if exists > 0 {
		log.Printf("[Notification] Duplicate message skipped: %s", event.MessageID)
		return nil
	}

	var sendErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := retryDelays[attempt-1]
			log.Printf("[Notification] Retry %d/%d for %s, waiting %s",
				attempt, maxAttempts-1, event.MessageID, delay)
			time.Sleep(delay)
		}

		sendErr = u.emailSender.Send(ctx, event)
		if sendErr == nil {
			break
		}
		log.Printf("[Notification] Attempt %d/%d failed for %s: %v",
			attempt+1, maxAttempts, event.MessageID, sendErr)
	}

	if sendErr != nil {
		return sendErr
	}

	if err := u.redisClient.Set(ctx, idempotencyKey, "1", idempotencyTTL).Err(); err != nil {
		log.Printf("[Notification] Failed to set idempotency key for %s: %v", event.MessageID, err)
	}

	u.mu.Lock()
	defer u.mu.Unlock()
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
