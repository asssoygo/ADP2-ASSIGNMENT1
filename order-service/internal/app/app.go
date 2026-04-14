package app

import (
	"database/sql"
	"log"

	"order-service/internal/repository/postgres"
	orderHTTP "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"order-service/pkg/grpcclient"

	"github.com/gin-gonic/gin"
)

func BuildRouter(db *sql.DB, paymentGRPCAddr string) *gin.Engine {
	orderRepo := postgres.NewOrderRepository(db)

	paymentClient, err := grpcclient.NewPaymentClient(paymentGRPCAddr)
	if err != nil {
		log.Fatal("failed to create grpc payment client: ", err)
	}

	orderUsecase := usecase.NewOrderUsecase(orderRepo, paymentClient)
	orderHandler := orderHTTP.NewOrderHandler(orderUsecase)

	return orderHTTP.NewRouter(orderHandler)
}
