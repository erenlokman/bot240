package storage

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/erenlokman/bot240/config"
	"github.com/erenlokman/bot240/models"
	"github.com/go-resty/resty/v2"
)

func FetchCryptoNews(chatID int64, db *sql.DB) {
	client := resty.New()
	authToken := config.GetEnvVar("CRYPTOPANIC_AUTH_TOKEN")
	response, err := client.R().
		SetQueryParam("auth_token", authToken).
		SetQueryParam("kind", "news").
		Get("https://cryptopanic.com/api/v1/posts/")
	if err != nil {
		log.Printf("Error fetching crypto news: %v", err)
		return
	}

	var data models.CryptoPanicResponse
	if err := json.Unmarshal(response.Body(), &data); err != nil {
		log.Printf("Error processing news data: %v", err)
		return
	}

	insertSQL := `INSERT INTO news (title, url, published_at) VALUES (?, ?, ?)`
	for _, item := range data.Results {
		_, err := db.Exec(insertSQL, item.Title, item.URL, time.Now())
		if err != nil {
			log.Printf("Failed to insert news item: %v", err)
		}
	}
}
