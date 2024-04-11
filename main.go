package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

var bot *tgbotapi.BotAPI

// CryptoNewsResponse struct holds the data structure for the response from the CryptoCompare API.
type CryptoNewsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		PublishedOn int64  `json:"published_on"`
		URL         string `json:"url"`
	} `json:"Data"`
}

// CryptoPanicResponse struct for parsing responses from the CryptoPanic API.
type CryptoPanicResponse struct {
	Results []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"results"`
}

// TradingViewAlert struct for handling webhook alerts from TradingView.
type TradingViewAlert struct {
	StrategyName string  `json:"strategyName"`
	Ticker       string  `json:"ticker"`
	Price        float64 `json:"price"`
	Message      string  `json:"message"`
}

type NewsAPIResponseArticle struct {
	Source struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"source"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	PublishedAt time.Time `json:"publishedAt"`
}

type NewsAPIResponse struct {
	Status       string                   `json:"status"`
	TotalResults int                      `json:"totalResults"`
	Articles     []NewsAPIResponseArticle `json:"articles"`
}

// Main function initializes the application.
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

func parseCommand(text string) (string, string) {
	parts := strings.Split(text, " ")
	if len(parts) > 1 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// getEnvVar retrieves environment variables and logs a fatal error if not set.
func getEnvVar(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is not set", key)
	}
	return value
}

// setupRoutes defines HTTP routes for the web server.
func setupRoutes() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ngrok is properly configured and the server is running."))
	})
	http.HandleFunc("/webhook", handleTradingViewWebhook)
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
}

// handleTradingViewWebhook processes POST requests from TradingView.
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

// sendMessageToTelegram sends a message to a specific Telegram chat without URL previews.
func sendMessageToTelegram(message string, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.DisableWebPagePreview = true // Ensure this is true
	log.Printf("Sending message with DisableWebPagePreview set to %v", msg.DisableWebPagePreview)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
	}
}

// handleUpdates listens for messages from Telegram and responds appropriately.
func handleUpdates(updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID
		command, ticker := parseCommand(update.Message.Text)

		switch command {
		case "/panic-news":
			db := initDB()
			defer db.Close()
			fetchCryptoNews(chatID, db, ticker) // Pass ticker
		case "/compare-news":
			fetchCryptoCompareNews(chatID, ticker) // Pass ticker
		case "/news":
			fetchNewsAPI(chatID, ticker) // Pass ticker
		case "/analyze":
			fetchNewsAPI(chatID, ticker)
		default:
			response := getOpenAIResponse(update.Message.Text)
			sendMessage(chatID, []string{response})
		}
	}
}

// fetchCryptoNews fetches crypto news from CryptoPanic and stores them in the database.
func fetchCryptoNews(chatID int64, db *sql.DB, ticker string) {
	client := resty.New()
	authToken := getEnvVar("CRYPTOPANIC_AUTH_TOKEN")
	req := client.R().
		SetQueryParam("auth_token", authToken).
		SetQueryParam("kind", "news")

	if ticker != "" {
		req.SetQueryParam("filter", ticker) // Assuming API supports a 'filter' parameter
	}

	response, err := req.Get("https://cryptopanic.com/api/v1/posts/")
	if err != nil {
		log.Printf("Error fetching crypto news: %v", err)
		sendMessage(chatID, []string{"Error fetching crypto news."})
		return
	}

	var data CryptoPanicResponse
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error processing news data: %v", err)
		sendMessage(chatID, []string{"Error processing news data."})
		return
	}

	// If the API does not support filtering, do it manually here
	if ticker != "" {
		filteredResults := []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		}{}
		for _, item := range data.Results {
			if strings.Contains(item.Title, ticker) { // Simple string matching; adjust as needed
				filteredResults = append(filteredResults, item)
			}
		}
		data.Results = filteredResults
	}

	insertSQL := `INSERT INTO news (title, url, published_at) VALUES (?, ?, ?)`
	for _, item := range data.Results {
		_, err := db.Exec(insertSQL, item.Title, item.URL, time.Now())
		if err != nil {
			log.Printf("Failed to insert news item: %v", err)
		}
	}

	sendMessage(chatID, []string{formatNewsResponse(data)})
}

// fetchCryptoCompareNews fetches news from CryptoCompare API.
func fetchCryptoCompareNews(chatID int64, ticker string) {
	client := resty.New()
	apiKey := getEnvVar("CRYPTOCOMPARE_API_KEY")
	response, err := client.R().
		SetQueryParam("api_key", apiKey).
		SetQueryParam("categories", "BTC,ETH"). // Optional: filter by categories
		Get("https://min-api.cryptocompare.com/data/v2/news/?lang=EN")
	if err != nil {
		log.Printf("Error fetching crypto compare news: %v", err)
		sendMessage(chatID, []string{"Error fetching news."})
		return
	}

	var data CryptoNewsResponse
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error processing news data: %v", err)
		sendMessage(chatID, []string{"Error processing news data."})
		return
	}

	// Manual filtering if ticker is specified and not supported by the API directly
	if ticker != "" {
		filteredData := []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Body        string `json:"body"`
			PublishedOn int64  `json:"published_on"`
			URL         string `json:"url"`
		}{}
		for _, item := range data.Data {
			// Here you could match against the title or body; this example matches the title
			if strings.Contains(strings.ToUpper(item.Title), strings.ToUpper(ticker)) {
				filteredData = append(filteredData, item)
			}
		}
		data.Data = filteredData
	}

	sendMessage(chatID, []string{formatNewsResponseFromCryptoCompare(data)})
}

// formatNewsResponse formats the news data from CryptoPanic for Telegram display.
func formatNewsResponse(data CryptoPanicResponse) string {
	newsMessage := "Latest Crypto News:\n"
	for _, item := range data.Results {
		newsMessage += fmt.Sprintf("%s\n%s\n\n", item.Title, item.URL)
	}
	return newsMessage
}

// formatNewsResponseFromCryptoCompare formats the news data from CryptoCompare for Telegram display.
func formatNewsResponseFromCryptoCompare(data CryptoNewsResponse) string {
	newsMessage := "Latest Market News:\n"
	for _, item := range data.Data {
		newsMessage += fmt.Sprintf("%s\n%s\n\n", item.Title, item.URL)
	}
	return newsMessage
}

// Fetches news from NewsAPI filtered by a specific ticker and from the last 30 minutes
func fetchNewsAPI(chatID int64, ticker string) {
	client := resty.New()
	apiKey := getEnvVar("NEWSAPI_API_KEY") // Make sure you have NEWSAPI_API_KEY in your environment variables

	// Calculate the time 30 minutes ago from now
	// fromTime := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

	// Construct the URL for the 'everything' endpoint instead of 'top-headlines'

	response, err := client.R().
		SetQueryParam("apiKey", apiKey).
		SetQueryParam("q", ticker). // Ensure the query is non-empty
		// SetQueryParam("from", fromTime).        // Articles published in the last 30 minutes
		SetQueryParam("sortBy", "publishedAt"). // Correct parameter for sorting by publication date
		SetQueryParam("pageSize", "1").
		Get("https://newsapi.org/v2/everything")

	if err != nil {
		log.Printf("Error fetching news from NewsAPI: %v", err)
		sendMessage(chatID, []string{"Error fetching news."})
		return
	}

	// print request URL
	log.Printf("Request URL: %v", response.Request.URL)

	// Define the struct for parsing the JSON response
	var data NewsAPIResponse

	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error processing news data from NewsAPI: %v", err)
		sendMessage(chatID, []string{"Error processing news data."})
		return
	}

	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error processing news data from NewsAPI: %v", err)
		sendMessage(chatID, []string{"Error processing news data."})
		return
	}

	if len(data.Articles) > 0 {
		analyzeNewsWithAI(chatID, data.Articles)
	} else {
		sendMessage(chatID, []string{"No relevant articles found for analysis."})
	}

	// Format the response into a readable message
	sendMessage(chatID, []string{formatNewsAPIResponse(data)})
}

// Function to format NewsAPI response
func formatNewsAPIResponse(data NewsAPIResponse) string {
	if data.TotalResults == 0 {
		return "No recent news found for the given topic."
	}

	newsMessage := "Latest News:\n"
	for _, article := range data.Articles {
		newsMessage += fmt.Sprintf("Title: %s\nAuthor: %s\nSource: %s\nPublished: %s\nURL: %s\n\n",
			article.Title, article.Author, article.Source.Name, article.PublishedAt.Format(time.RFC1123), article.URL)
	}
	return newsMessage
}

// getOpenAIResponse sends a prompt to OpenAI and returns the response.
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

func analyzeNewsWithAI(chatID int64, articles []NewsAPIResponseArticle) {
	// Positive and negative keywords
	positiveKeywords := []string{"growth", "upward", "bullish", "surge", "rally", "record high", "advancing", "gains", "profit", "outperform"}
	negativeKeywords := []string{"ban", "hack", "crash", "plunge", "downward", "bearish", "losses", "decline", "sell-off", "underperform"}

	// Iterate through each article to analyze
	for _, article := range articles {
		prompt := fmt.Sprintf("Analyze the sentiment of this news article titled '%s': %s", article.Title, article.Description)
		sentiment := getOpenAIResponse(prompt)
		decision := makeTradingDecision([]string{sentiment}, positiveKeywords, negativeKeywords) // Convert sentiment to a string slice
		sendMessage(chatID, []string{fmt.Sprintf("Article: %s\nSentiment: %s\nDecision: %s\nURL: %s\n", article.Title, sentiment, decision, article.URL)})
	}
}

func makeTradingDecision(sentiment, positiveKeywords, negativeKeywords []string) string {
	lowerSentiment := strings.ToLower(strings.Join(sentiment, " ")) // Join the elements of the sentiment slice into a single string
	for _, keyword := range positiveKeywords {
		if strings.Contains(lowerSentiment, keyword) {
			return "Buy"
		}
	}
	for _, keyword := range negativeKeywords {
		if strings.Contains(lowerSentiment, keyword) {
			return "Sell"
		}
	}
	return "Hold" // Default decision if no keywords match
}

// extractOpenAIResponse extracts the actual response from the OpenAI JSON response.
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

// // sendMessage sends multiple messages to a specific Telegram chat if the content is too long.
func sendMessage(chatID int64, messages []string) {
	const maxLength = 4096
	for _, message := range messages {
		for len(message) > 0 {
			if len(message) > maxLength {
				// Find the last newline character within the maxLength
				splitIndex := strings.LastIndex(message[:maxLength], "\n")
				if splitIndex == -1 {
					splitIndex = maxLength
				}
				part := message[:splitIndex]
				msg := tgbotapi.NewMessage(chatID, part)
				msg.DisableWebPagePreview = true // Ensure this is true
				bot.Send(msg)
				message = message[splitIndex:]
			} else {
				msg := tgbotapi.NewMessage(chatID, message)
				msg.DisableWebPagePreview = true // Ensure this is true
				bot.Send(msg)
				break
			}
		}
	}
}

// initDB initializes the SQLite database and creates the news table if it doesn't exist.
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
