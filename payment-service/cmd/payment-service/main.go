package main

import (
	"database/sql"
	"log"
	"net"
	"os"
	"payment-service/internal/app"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := getEnv("PAYMENT_DB_URL", "postgres://postgres:123@localhost:5434/payment_db?sslmode=disable")
	httpPort := getEnv("PORT", "8081")
	grpcPort := getEnv("GRPC_PORT", "50051")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("failed to open db: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping db: ", err)
	}

	router, grpcServer := app.BuildServers(db)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatal("failed to listen grpc: ", err)
	}

	go func() {
		log.Println("payment-service grpc running on port " + grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal("failed to run grpc server: ", err)
		}
	}()

	log.Println("payment-service http running on port " + httpPort)
	if err := router.Run(":" + httpPort); err != nil {
		log.Fatal("failed to run http server: ", err)
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
