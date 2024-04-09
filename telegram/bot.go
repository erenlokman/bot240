package telegram

import (
	"log"

	"github.com/erenlokman/bot240/storage"
	"github.com/erenlokman/bot240/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var bot *tgbotapi.BotAPI // Ensure this is the bot instance being used globally.

func NewBot(token string) error {
	var err error
	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}
	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)
	return nil
}

func HandleUpdates() {
	if bot == nil {
		log.Fatal("Bot instance is nil. Make sure to initialize it before handling updates.")
	}
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

		// print the CHAT ID and the message
		log.Printf("[%d] %s", update.Message.Chat.ID, update.Message.Text)

		switch update.Message.Text {
		case "/panic-news":
			db, err := storage.InitDB()
			if err != nil {
				log.Printf("Failed to initialize DB: %v", err)
				continue
			}
			defer db.Close()
			storage.FetchCryptoNews(update.Message.Chat.ID, db)
		default:
			response := utils.GetOpenAIResponse(update.Message.Text)
			SendMessageToTelegram(response, update.Message.Chat.ID)
		}
	}
}

func SendMessageToTelegram(message string, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, message)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Failed to send message to Telegram: %v", err)
	}
}
