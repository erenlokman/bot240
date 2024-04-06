package main

import (
	"log"
	"net/http"

	"https://github.com/erenlokman/bot240/api"
	"https://github.com/erenlokman/bot240/config"
	"https://github.com/erenlokman/bot240/storage"
	"https://github.com/erenlokman/bot240/telegram"
)

func main() {
	if err := config.LoadEnv(); err != nil {
		log.Fatal("Error loading .env file")
	}

	bot, err := telegram.NewBotAPI(config.GetEnvVar("TELEGRAM_BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	db := storage.InitDB()
	defer db.Close()

	api.SetupRoutes(bot, db)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
