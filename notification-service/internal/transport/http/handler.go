package http

import (
	"encoding/json"
	"net/http"

	"notification-service/internal/usecase"
)

func NewHandler(uc *usecase.NotificationUsecase) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		notifications := uc.GetRecentNotifications()
		if notifications == nil {
			notifications = []usecase.ProcessedNotification{}
		}
		json.NewEncoder(w).Encode(notifications)
	})

	return mux
}
