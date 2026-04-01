package main

import (
	"database/sql"
	"log"
	"order-service/internal/app"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := getEnv("ORDER_DB_URL", "postgres://postgres:123@localhost:5433/order_db?sslmode=disable")
	paymentBaseURL := getEnv("PAYMENT_SERVICE_URL", "http://localhost:8081")
	port := getEnv("PORT", "8080")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("failed to open db: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping db: ", err)
	}

	router := app.BuildRouter(db, paymentBaseURL)

	log.Println("order-service running on port " + port)
	if err := router.Run(":" + port); err != nil {
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
