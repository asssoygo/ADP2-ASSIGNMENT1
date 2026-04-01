package app

import (
	"database/sql"
	"order-service/internal/repository/postgres"
	orderHTTP "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"order-service/pkg/httpclient"

	"github.com/gin-gonic/gin"
)

func BuildRouter(db *sql.DB, paymentBaseURL string) *gin.Engine {
	orderRepo := postgres.NewOrderRepository(db)
	paymentClient := httpclient.NewPaymentClient(paymentBaseURL)
	orderUsecase := usecase.NewOrderUsecase(orderRepo, paymentClient)
	orderHandler := orderHTTP.NewOrderHandler(orderUsecase)

	return orderHTTP.NewRouter(orderHandler)
}
