package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"https://github.com/erenlokman/bot240/telegram"
)

func SetupRoutes(bot *telegram.BotAPI, db *sql.DB) {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/webhook", makeWebhookHandler(bot, db))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Server is running")
}

func makeWebhookHandler(bot *telegram.BotAPI, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Handle webhook
	}
}
