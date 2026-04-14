package http

import (
	"errors"
	"net/http"
	"order-service/internal/stream"
	"order-service/internal/usecase"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	usecase *usecase.OrderUsecase
	streams *stream.Manager
}

func NewOrderHandler(usecase *usecase.OrderUsecase, streams *stream.Manager) *OrderHandler {
	return &OrderHandler{
		usecase: usecase,
		streams: streams,
	}
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id"`
	ItemName   string `json:"item_name"`
	Amount     int64  `json:"amount"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	order, err := h.usecase.CreateOrder(req.CustomerID, req.ItemName, req.Amount)
	if err != nil {
		if err.Error() == "payment service unavailable" ||
			len(err.Error()) >= 27 && err.Error()[:27] == "payment service unavailable" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.streams.Publish(order.ID, order.Status)
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.usecase.GetOrder(id)
	if err != nil {
		if errors.Is(err, errors.New("order not found")) || err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.usecase.CancelOrder(id)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.streams.Publish(order.ID, order.Status)
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")

	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	order, err := h.usecase.UpdateOrderStatus(id, req.Status)
	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.streams.Publish(order.ID, order.Status)
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) GetOrdersByAmountRange(c *gin.Context) {
	minAmountStr := c.Query("min_amount")
	maxAmountStr := c.Query("max_amount")

	if minAmountStr == "" || maxAmountStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_amount and max_amount are required"})
		return
	}

	minAmount, err := strconv.ParseInt(minAmountStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "min_amount must be a valid integer"})
		return
	}

	maxAmount, err := strconv.ParseInt(maxAmountStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_amount must be a valid integer"})
		return
	}

	orders, err := h.usecase.GetOrdersByAmountRange(minAmount, maxAmount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}
