package app

import (
	"database/sql"
	"log"
	"os"

	"contracts/paymentpb"
	"payment-service/internal/infrastructure/rabbitmq"
	"payment-service/internal/repository/postgres"
	paymentGRPC "payment-service/internal/transport/grpc"
	paymentHTTP "payment-service/internal/transport/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func BuildServers(db *sql.DB) (*gin.Engine, *grpc.Server) {
	paymentRepo := postgres.NewPaymentRepository(db)
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	publisher, err := rabbitmq.NewPublisher(rabbitURL)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}

	paymentUsecase := usecase.NewPaymentUsecase(paymentRepo, publisher)

	paymentHandler := paymentHTTP.NewPaymentHandler(paymentUsecase)
	router := paymentHTTP.NewRouter(paymentHandler)

	grpcServer := grpc.NewServer()
	paymentpb.RegisterPaymentServiceServer(
		grpcServer,
		paymentGRPC.NewPaymentServer(paymentUsecase),
	)

	return router, grpcServer
}
