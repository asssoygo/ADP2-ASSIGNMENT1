package app

import (
	"net/http"

	"notification-service/internal/infrastructure/rabbitmq"
	notifHTTP "notification-service/internal/transport/http"
	"notification-service/internal/usecase"
)

type App struct {
	Consumer   *rabbitmq.Consumer
	HTTPServer *http.Server
}

func NewApp(rabbitURL, httpPort string) (*App, error) {
	notifUsecase := usecase.NewNotificationUsecase()

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
