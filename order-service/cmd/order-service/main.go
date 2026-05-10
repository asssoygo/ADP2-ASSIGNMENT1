package main

import (
	"database/sql"
	"log"
	"net"
	"order-service/internal/app"
	"os"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	dbURL := getEnv("ORDER_DB_URL", "postgres://postgres:123@localhost:5433/order_db?sslmode=disable")
	paymentGRPCAddr := getEnv("PAYMENT_GRPC_ADDR", "localhost:50051")
	httpPort := getEnv("PORT", "8080")
	grpcPort := getEnv("ORDER_GRPC_PORT", "50052")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("failed to open db: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping db: ", err)
	}

	redisOpt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("failed to parse redis URL: ", err)
	}
	redisClient := redis.NewClient(redisOpt)

	router, grpcServer := app.BuildServers(db, paymentGRPCAddr, redisClient)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal("failed to listen grpc: ", err)
	}

	go func() {
		log.Println("order-service grpc running on port " + grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to run grpc server: ", err)
		}
	}()

	log.Println("order-service http running on port " + httpPort)
	if err := router.Run(":" + httpPort); err != nil {
		log.Fatal("failed to run server: ", err)
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
