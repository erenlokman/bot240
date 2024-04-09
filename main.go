package main

import (
	"log"

	"github.com/erenlokman/bot240/api"
	"github.com/erenlokman/bot240/config"
	"github.com/erenlokman/bot240/telegram"
)

func main() {
	if err := config.LoadEnv(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	if err := telegram.NewBot(config.GetEnvVar("TELEGRAM_BOT_TOKEN")); err != nil {
		log.Fatal("Failed to initialize bot:", err)
	}

	telegram.HandleUpdates()
	api.SetupRoutes()
}
