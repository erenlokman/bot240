package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var bot *tgbotapi.BotAPI
var chatID int64 // Replace with your chat ID

type CryptoNewsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		PublishedOn int64  `json:"published_on"`
		URL         string `json:"url"`
	} `json:"Data"`
}

type CryptoPanicResponse struct {
	Results []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"results"`
}

type TradingViewAlert struct {
	StrategyName string  `json:"strategyName"`
	Ticker       string  `json:"ticker"`
	Price        float64 `json:"price"`
	Message      string  `json:"message"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	var err error
	botToken := getEnvVar("TELEGRAM_BOT_TOKEN")
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	setupRoutes()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	handleUpdates(updates)
}

func getEnvVar(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is not set", key)
	}
	return value
}

func setupRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ngrok is properly configured and the server is running."))
	})
	http.HandleFunc("/webhook", handleTradingViewWebhook)
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
}

func handleTradingViewWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// Logging the body to see what you're receiving
	log.Printf("Received webhook payload: %s", string(body))

	sendMessageToTelegram(string(body), 1520870444) // Send the raw text to Telegram

}

func sendMessageToTelegram(message string, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, message)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
	}
}

func handleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		switch update.Message.Text {
		case "/panic-news":
			db := initDB()
			defer db.Close()
			fetchCryptoNews(chatID, db)
		case "/compare-news":
			fetchCryptoCompareNews(chatID)
		default:
			response := getOpenAIResponse(update.Message.Text)
			sendMessage(update.Message.Chat.ID, response)
		}
	}
}

func fetchCryptoNews(chatID int64, db *sql.DB) {
	client := resty.New()
	authToken := getEnvVar("CRYPTOPANIC_AUTH_TOKEN")
	response, err := client.R().
		SetQueryParam("auth_token", authToken).
		SetQueryParam("kind", "news").
		Get("https://cryptopanic.com/api/v1/posts/")
	if err != nil {
		log.Printf("Error fetching crypto news: %v", err)
		sendMessage(chatID, "Error fetching crypto news.")
		return
	}

	var data CryptoPanicResponse
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error processing news data: %v", err)
		sendMessage(chatID, "Error processing news data.")
		return
	}

	insertSQL := `INSERT INTO news (title, url, published_at) VALUES (?, ?, ?)`
	for _, item := range data.Results {
		_, err := db.Exec(insertSQL, item.Title, item.URL, time.Now())
		if err != nil {
			log.Printf("Failed to insert news item: %v", err)
		}
	}

	sendMessage(chatID, formatNewsResponse(data))
}

func fetchCryptoCompareNews(chatID int64) {
	client := resty.New()
	apiKey := getEnvVar("CRYPTOCOMPARE_API_KEY")
	response, err := client.R().
		SetQueryParam("api_key", apiKey).
		Get("https://min-api.cryptocompare.com/data/v2/news/?lang=EN")
	if err != nil {
		log.Printf("Error fetching crypto compare news: %v", err)
		sendMessage(chatID, "Error fetching crypto compare news.")
		return
	}

	var data CryptoNewsResponse
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error processing news data: %v", err)
		sendMessage(chatID, "Error processing news data.")
		return
	}

	sendMessage(chatID, formatNewsResponseFromCryptoCompare(data))
}

func formatNewsResponse(data CryptoPanicResponse) string {
	newsMessage := "Latest Crypto News:\n"
	for _, item := range data.Results {
		newsMessage += fmt.Sprintf("%s\n%s\n\n", item.Title, item.URL)
	}
	return newsMessage
}

func formatNewsResponseFromCryptoCompare(data CryptoNewsResponse) string {
	newsMessage := "Latest Market News:\n"
	for _, item := range data.Data {
		newsMessage += fmt.Sprintf("%s\n%s\n\n", item.Title, item.URL)
	}
	return newsMessage
}

func getOpenAIResponse(prompt string) string {
	client := resty.New()
	apiKey := getEnvVar("OPENAI_API_KEY")

	body := map[string]interface{}{
		"model": "gpt-4-0125-preview",
		"messages": []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": prompt,
			},
		},
	}

	response, err := client.R().
		SetAuthToken(apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post("https://api.openai.com/v1/chat/completions")

	if err != nil {
		log.Printf("Error requesting OpenAI: %v", err)
		return "Error communicating with OpenAI."
	}

	var data map[string]interface{}
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error decoding OpenAI response: %v", err)
		return "Error decoding OpenAI response."
	}

	return extractOpenAIResponse(data)
}

func extractOpenAIResponse(data map[string]interface{}) string {
	if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
		if firstChoice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := firstChoice["message"].(map[string]interface{}); ok {
				return message["content"].(string)
			}
		}
	}
	return "Failed to extract response."
}

func sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./crypto_news.db")
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS news (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        url TEXT NOT NULL,
        published_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`

	if _, err = db.Exec(createTableSQL); err != nil {
		log.Fatal("Error creating table: ", err)
	}

	return db
}
