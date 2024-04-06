package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"io/ioutil"
	"net/http"

	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

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

// Define a struct that matches the expected JSON payload structure from TradingView
type TradingViewAlert struct {
	StrategyName string  `json:"strategyName"`
	Ticker       string  `json:"ticker"`
	Price        float64 `json:"price"`
	Message      string  `json:"message"`
}

func handleTradingViewWebhook(bot *tgbotapi.BotAPI, w http.ResponseWriter, r *http.Request, chatID int64) {
	if r.Method != "POST" {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// Deserialize the JSON body into the TradingViewAlert struct
	var alert TradingViewAlert
	if err := json.Unmarshal(body, &alert); err != nil {
		log.Printf("Error parsing webhook payload: %v", err)
		http.Error(w, "Error parsing payload", http.StatusBadRequest)
		return
	}

	log.Printf("Received TradingView alert: %+v", alert)

	// Call function to send the alert via Telegram
	sendMessageToTelegram(bot, alert, chatID)
}

func sendMessageToTelegram(bot *tgbotapi.BotAPI, alert TradingViewAlert, chatID int64) {
	msgText := fmt.Sprintf("Alert from TradingView!\nStrategy: %s\nTicker: %s\nPrice: %.2f\nMessage: %s",
		alert.StrategyName, alert.Ticker, alert.Price, alert.Message)

	msg := tgbotapi.NewMessage(chatID, msgText)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
	}
}

func setupRoutes(bot *tgbotapi.BotAPI, chatID int64) {
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		handleTradingViewWebhook(bot, w, r, chatID)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getEnvVar(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is not set", key)
	}
	return value
}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./crypto_news.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS news (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT NOT NULL,
        url TEXT NOT NULL,
        published_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func fetchCryptoNews(bot *tgbotapi.BotAPI, chatID int64, db *sql.DB) {
	client := resty.New()
	authToken := getEnvVar("CRYPTOPANIC_AUTH_TOKEN")

	response, err := client.R().
		SetQueryParam("auth_token", authToken).
		SetQueryParam("kind", "news").
		Get("https://cryptopanic.com/api/v1/posts/")
	if err != nil {
		log.Printf("Error fetching crypto news: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Error fetching crypto news."))
		return
	}

	var data CryptoPanicResponse
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error processing news data: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Error processing news data."))
		return
	}

	insertSQL := `INSERT INTO news (title, url, published_at) VALUES (?, ?, ?)`
	for _, item := range data.Results {
		_, err := db.Exec(insertSQL, item.Title, item.URL, time.Now())
		if err != nil {
			log.Printf("Failed to insert news item: %v", err)
		}
	}

	newsMessage := formatNewsResponse(data)
	bot.Send(tgbotapi.NewMessage(chatID, newsMessage))
}

func fetchCryptoCompareNews(bot *tgbotapi.BotAPI, chatID int64) {
	client := resty.New()
	apiKey := getEnvVar("CRYPTOCOMPARE_API_KEY")

	response, err := client.R().
		SetQueryParam("api_key", apiKey).
		Get("https://min-api.cryptocompare.com/data/v2/news/?lang=EN")
	if err != nil {
		log.Fatalf("Error fetching crypto news: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Error fetching crypto news."))
		return
	}

	var data CryptoNewsResponse
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Fatalf("Error processing news data: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Error processing news data."))
		return
	}

	for _, item := range data.Data {
		fmt.Printf("%s\n%s\n\n", item.Title, item.URL)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("%s\n%s", item.Title, item.URL)))

	}
}

func formatNewsResponse(data CryptoPanicResponse) string {
	newsMessage := "Latest Crypto News:\n"
	for _, item := range data.Results {
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
		log.Fatalf("Error requesting OpenAI: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Fatalf("Error decoding OpenAI response: %v", err)
	}

	return extractOpenAIResponse(data)
}

func extractOpenAIResponse(data map[string]interface{}) string {
	choices := data["choices"].([]interface{})
	firstChoice := choices[0].(map[string]interface{})
	messages := firstChoice["message"].(map[string]interface{})
	return messages["content"].(string)
}

func main() {

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	botToken := getEnvVar("TELEGRAM_BOT_TOKEN")
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.Text == "/panic-news" {
			fetchCryptoNews(bot, update.Message.Chat.ID, nil)
			continue
		}

		if update.Message.Text == "/compare-news" {
			fetchCryptoCompareNews(bot, update.Message.Chat.ID)
			continue
		}

		if update.Message.Text == "/store-data" {
			db := initDB()
			defer db.Close()

			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				fetchCryptoNews(bot, update.Message.Chat.ID, db)
			}
		}

		openAIResponse := getOpenAIResponse(update.Message.Text)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, openAIResponse)
		msg.ReplyToMessageID = update.Message.MessageID

		setupRoutes(bot, update.Message.Chat.ID)

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
