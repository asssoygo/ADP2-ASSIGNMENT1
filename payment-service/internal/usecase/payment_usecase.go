package usecase

import (
	"errors"
	"fmt"
	"log"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUsecase struct {
	repo      PaymentRepository
	publisher EventPublisher
}

func NewPaymentUsecase(repo PaymentRepository, publisher EventPublisher) *PaymentUsecase {
	return &PaymentUsecase{repo: repo, publisher: publisher}
}

func (u *PaymentUsecase) CreatePayment(orderID string, amount int64, customerEmail string) (*domain.Payment, error) {
	if orderID == "" {
		return nil, errors.New("order_id is required")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	status := "Authorized"
	transactionID := uuid.New().String()

	if amount > 100000 {
		status = "Declined"
	}

	if customerEmail == "" {
		customerEmail = fmt.Sprintf("customer-%s@example.com", orderID)
	}

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        status,
		CustomerEmail: customerEmail,
	}

	if err := u.repo.Create(payment); err != nil {
		return nil, err
	}

	if u.publisher != nil {
		event := PaymentCompletedEvent{
			MessageID:     payment.ID,
			OrderID:       payment.OrderID,
			Amount:        payment.Amount,
			CustomerEmail: payment.CustomerEmail,
			Status:        payment.Status,
		}
		if err := u.publisher.PublishPaymentCompleted(event); err != nil {
			log.Printf("warning: failed to publish payment event: %v", err)
		}
	}

	return payment, nil
}

func (u *PaymentUsecase) GetPayment(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}

func (u *PaymentUsecase) ListPayments(status string) ([]domain.Payment, error) {
	if status == "" {
		return nil, errors.New("status is required")
	}

	if status != "Authorized" && status != "Declined" {
		return nil, errors.New("status must be Authorized or Declined")
	}

	return u.repo.ListByStatus(status)
}
