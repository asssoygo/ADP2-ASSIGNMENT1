package app

import (
	"database/sql"
	"payment-service/internal/repository/postgres"
	paymentHTTP "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

func BuildRouter(db *sql.DB) *gin.Engine {
	paymentRepo := postgres.NewPaymentRepository(db)
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo)
	paymentHandler := paymentHTTP.NewPaymentHandler(paymentUsecase)

	return paymentHTTP.NewRouter(paymentHandler)
}
