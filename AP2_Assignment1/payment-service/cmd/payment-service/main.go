package main

import (
	"database/sql"
	"log"
	"os"
	"payment-service/internal/app"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := getEnv("PAYMENT_DB_URL", "postgres://postgres:123@localhost:5434/payment_db?sslmode=disable")
	port := getEnv("PORT", "8081")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("failed to open db: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping db: ", err)
	}

	router := app.BuildRouter(db)

	log.Println("payment-service running on port " + port)
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
