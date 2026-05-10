package usecase

import (
	"context"
	"errors"
	"fmt"
	"order-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

const cacheTTL = 5 * time.Minute

type OrderUsecase struct {
	repo          OrderRepository
	cache         CacheRepository
	paymentClient PaymentClient
}

func NewOrderUsecase(repo OrderRepository, cache CacheRepository, paymentClient PaymentClient) *OrderUsecase {
	return &OrderUsecase{
		repo:          repo,
		cache:         cache,
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
		u.invalidateCache(order.ID)
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

	u.invalidateCache(order.ID)
	return u.repo.GetByID(order.ID)
}

func (u *OrderUsecase) GetOrder(id string) (*domain.Order, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("order:%s", id)

	if order, err := u.cache.Get(ctx, cacheKey); err == nil {
		return order, nil
	}

	order, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	_ = u.cache.Set(ctx, cacheKey, order, cacheTTL)
	return order, nil
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

	u.invalidateCache(id)
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

func (u *OrderUsecase) UpdateOrderStatus(id string, newStatus string) (*domain.Order, error) {
	if id == "" {
		return nil, errors.New("order_id is required")
	}
	if newStatus == "" {
		return nil, errors.New("status is required")
	}

	if _, err := u.repo.GetByID(id); err != nil {
		return nil, err
	}

	if err := u.repo.UpdateStatus(id, newStatus); err != nil {
		return nil, err
	}

	u.invalidateCache(id)
	return u.repo.GetByID(id)
}

func (u *OrderUsecase) invalidateCache(id string) {
	_ = u.cache.Delete(context.Background(), fmt.Sprintf("order:%s", id))
}
