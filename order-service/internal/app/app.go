package app

import (
	"database/sql"
	"log"

	"order-service/internal/repository/postgres"
	"order-service/internal/stream"
	orderGRPC "order-service/internal/transport/grpc"
	orderHTTP "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"order-service/pkg/grpcclient"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func BuildServers(db *sql.DB, paymentGRPCAddr string) (*gin.Engine, *grpc.Server) {
	orderRepo := postgres.NewOrderRepository(db)

	paymentClient, err := grpcclient.NewPaymentClient(paymentGRPCAddr)
	if err != nil {
		log.Fatal("failed to create grpc payment client: ", err)
	}

	orderUsecase := usecase.NewOrderUsecase(orderRepo, paymentClient)
	streams := stream.NewManager()

	orderHandler := orderHTTP.NewOrderHandler(orderUsecase, streams)
	router := orderHTTP.NewRouter(orderHandler)

	grpcServer := grpc.NewServer()
	orderGRPC.RegisterOrderGRPCServer(grpcServer, orderUsecase, streams)

	return router, grpcServer
}
