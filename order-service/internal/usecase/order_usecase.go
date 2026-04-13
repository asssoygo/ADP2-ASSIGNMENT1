package usecase

import (
	"errors"
	"order-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

type OrderUsecase struct {
	repo          OrderRepository
	paymentClient PaymentClient
}

func NewOrderUsecase(repo OrderRepository, paymentClient PaymentClient) *OrderUsecase {
	return &OrderUsecase{
		repo:          repo,
		paymentClient: paymentClient,
	}
}

func (u *OrderUsecase) CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error) {
	if customerID == "" || itemName == "" {
		return nil, errors.New("customer_id and item_name are required")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}

	order := &domain.Order{
		ID:         uuid.New().String(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     "Pending",
		CreatedAt:  time.Now(),
	}

	if err := u.repo.Create(order); err != nil {
		return nil, err
	}

	paymentStatus, err := u.paymentClient.CreatePayment(order.ID, order.Amount)
	if err != nil {
		_ = u.repo.UpdateStatus(order.ID, "Failed")
		return nil, err
	}

	if paymentStatus == "Authorized" {
		order.Status = "Paid"
	} else {
		order.Status = "Failed"
	}

	if err := u.repo.UpdateStatus(order.ID, order.Status); err != nil {
		return nil, err
	}

	return u.repo.GetByID(order.ID)
}

func (u *OrderUsecase) GetOrder(id string) (*domain.Order, error) {
	return u.repo.GetByID(id)
}

func (u *OrderUsecase) CancelOrder(id string) (*domain.Order, error) {
	order, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if order.Status != "Pending" {
		return nil, errors.New("only pending orders can be cancelled")
	}

	if err := u.repo.UpdateStatus(id, "Cancelled"); err != nil {
		return nil, err
	}

	return u.repo.GetByID(id)
}

func (u *OrderUsecase) GetOrdersByAmountRange(minAmount, maxAmount int64) ([]domain.Order, error) {
	if minAmount < 0 {
		return nil, errors.New("min_amount must be greater than or equal to 0")
	}

	if maxAmount > 100000 {
		return nil, errors.New("max_amount must be less than or equal to 100000")
	}

	if minAmount > maxAmount {
		return nil, errors.New("min_amount must not be greater than max_amount")
	}

	return u.repo.GetByAmountRange(minAmount, maxAmount)
}
