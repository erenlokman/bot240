package api

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/erenlokman/bot240/models"
	"github.com/erenlokman/bot240/telegram"
)

func HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Server is running."))
}

func HandleTradingViewWebhook(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received TradingView webhook request from %s", r.RemoteAddr)
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var alert models.TradingViewAlert
	if err := json.Unmarshal(body, &alert); err != nil {
		log.Printf("Error parsing TradingView alert: %v", err)
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Received webhook payload: %+v", alert)
	telegram.SendMessageToTelegram("Received TradingView alert: "+alert.Message, 1520870444) // Placeholder chat ID

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Webhook received"))
}
