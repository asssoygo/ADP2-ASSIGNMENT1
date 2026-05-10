package app

import (
	"log"
	"net/http"

	"notification-service/internal/infrastructure/email"
	"notification-service/internal/infrastructure/rabbitmq"
	notifHTTP "notification-service/internal/transport/http"
	"notification-service/internal/usecase"

	"github.com/redis/go-redis/v9"
)

type App struct {
	Consumer   *rabbitmq.Consumer
	HTTPServer *http.Server
}

func NewApp(rabbitURL, httpPort, redisURL, providerMode string) (*App, error) {
	redisOpt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("failed to parse redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpt)

	var emailSender usecase.EmailSender
	if providerMode == "REAL" {
		log.Println("[Notification] REAL provider not implemented, using simulated")
		emailSender = email.NewSimulatedEmailSender()
	} else {
		emailSender = email.NewSimulatedEmailSender()
	}

	notifUsecase := usecase.NewNotificationUsecase(emailSender, redisClient)

	consumer, err := rabbitmq.NewConsumer(rabbitURL, notifUsecase)
	if err != nil {
		return nil, err
	}

	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: notifHTTP.NewHandler(notifUsecase),
	}

	return &App{Consumer: consumer, HTTPServer: httpServer}, nil
}
