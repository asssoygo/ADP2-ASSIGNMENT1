package usecase

import (
	"errors"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUsecase struct {
	repo PaymentRepository
}

func NewPaymentUsecase(repo PaymentRepository) *PaymentUsecase {
	return &PaymentUsecase{repo: repo}
}

func (u *PaymentUsecase) CreatePayment(orderID string, amount int64) (*domain.Payment, error) {
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

	payment := &domain.Payment{
		ID:            uuid.New().String(),
		OrderID:       orderID,
		TransactionID: transactionID,
		Amount:        amount,
		Status:        status,
	}

	if err := u.repo.Create(payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (u *PaymentUsecase) GetPayment(orderID string) (*domain.Payment, error) {
	return u.repo.GetByOrderID(orderID)
}
