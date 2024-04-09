package api

import (
	"log"
	"net/http"
)

func SetupRoutes() {
	http.HandleFunc("/", HandleRoot)
	http.HandleFunc("/webhook", HandleTradingViewWebhook)
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
}
