package app

import (
	"database/sql"
	"log"

	"order-service/internal/infrastructure/cache"
	"order-service/internal/repository/postgres"
	"order-service/internal/stream"
	orderGRPC "order-service/internal/transport/grpc"
	orderHTTP "order-service/internal/transport/http"
	"order-service/internal/usecase"
	"order-service/pkg/grpcclient"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func BuildServers(db *sql.DB, paymentGRPCAddr string, redisClient *redis.Client) (*gin.Engine, *grpc.Server) {
	orderRepo := postgres.NewOrderRepository(db)
	cacheRepo := cache.NewRedisCache(redisClient)

	paymentClient, err := grpcclient.NewPaymentClient(paymentGRPCAddr)
	if err != nil {
		log.Fatal("failed to create grpc payment client: ", err)
	}

	orderUsecase := usecase.NewOrderUsecase(orderRepo, cacheRepo, paymentClient)
	streams := stream.NewManager()

	orderHandler := orderHTTP.NewOrderHandler(orderUsecase, streams)
	router := orderHTTP.NewRouter(orderHandler, redisClient)

	grpcServer := grpc.NewServer()
	orderGRPC.RegisterOrderGRPCServer(grpcServer, orderUsecase, streams)

	return router, grpcServer
}
