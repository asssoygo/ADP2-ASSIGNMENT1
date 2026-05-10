package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/app"
)

func main() {
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	httpPort := getEnv("HTTP_PORT", "8082")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	providerMode := getEnv("PROVIDER_MODE", "SIMULATED")

	application, err := app.NewApp(rabbitURL, httpPort, redisURL, providerMode)
	if err != nil {
		log.Fatalf("failed to start notification service: %v", err)
	}
	defer application.Consumer.Close()

	go func() {
		if err := application.Consumer.Start(); err != nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()

	go func() {
		log.Printf("[Notification] HTTP server listening on :%s", httpPort)
		if err := application.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Println("[Notification] Service started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[Notification] Shutting down...")
	application.HTTPServer.Close()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
