package app

import (
	"database/sql"

	"contracts/paymentpb"
	"payment-service/internal/repository/postgres"
	paymentGRPC "payment-service/internal/transport/grpc"
	paymentHTTP "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func BuildServers(db *sql.DB) (*gin.Engine, *grpc.Server) {
	paymentRepo := postgres.NewPaymentRepository(db)
	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo)

	paymentHandler := paymentHTTP.NewPaymentHandler(paymentUsecase)
	router := paymentHTTP.NewRouter(paymentHandler)

	grpcServer := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(
		grpcServer,
		paymentGRPC.NewPaymentServer(paymentUsecase),
	)

	return router, grpcServer
}
