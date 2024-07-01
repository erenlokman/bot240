# Crypto News Telegram Bot

This project is a Telegram bot designed to fetch and analyze cryptocurrency news from various sources, including CryptoPanic, CryptoCompare, and NewsAPI. Additionally, it processes webhook alerts from TradingView and sends them to a specified Telegram chat.

## Features

- Fetches and analyzes news from CryptoPanic, CryptoCompare, and NewsAPI.
- Processes webhook alerts from TradingView and sends them to Telegram.
- Analyzes the sentiment of news articles using OpenAI's GPT-4 API and makes trading decisions based on keywords.
- Stores news articles in an SQLite database.

## Prerequisites

- Go (version 1.18 or later)
- A Telegram bot token
- API keys for CryptoPanic, CryptoCompare, NewsAPI, and OpenAI
- SQLite3

## Installation

1. Clone the repository:
   git clone https://github.com/yourusername/crypto-news-telegram-bot.git
   cd crypto-news-telegram-bot

2. Create a `.env` file in the project root directory and add your API keys and bot token:
   TELEGRAM_BOT_TOKEN=your_telegram_bot_token
   CRYPTOPANIC_AUTH_TOKEN=your_cryptopanic_auth_token
   CRYPTOCOMPARE_API_KEY=your_cryptocompare_api_key
   NEWSAPI_API_KEY=your_newsapi_api_key
   OPENAI_API_KEY=your_openai_api_key

3. Build and run the application:
   go build -o crypto-news-bot
   ./crypto-news-bot

## Usage

### Telegram Commands

- `/panic-news [ticker]`: Fetches news from CryptoPanic, optionally filtered by a ticker.
- `/compare-news [ticker]`: Fetches news from CryptoCompare, optionally filtered by a ticker.
- `/news [ticker]`: Fetches news from NewsAPI, filtered by a ticker.
- `/analyze [ticker]`: Fetches and analyzes news from NewsAPI, filtered by a ticker.

### Webhook Setup

To handle TradingView webhooks, configure TradingView to send alerts to:
http://your_server_address:8080/webhook

## Database

The bot uses an SQLite database to store fetched news articles. The database file is named `crypto_news.db` and is created in the project root directory.

## Code Structure

- `main.go`: The main entry point of the application. Initializes the bot, sets up routes, and handles updates.
- `news.go`: Functions to fetch news from various APIs and store them in the database.
- `tradingview.go`: Handles webhook alerts from TradingView.
- `openai.go`: Integrates with OpenAI's GPT-4 API for sentiment analysis.
- `database.go`: Initializes the SQLite database and creates necessary tables.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [Resty](https://github.com/go-resty/resty)
- [GoDotEnv](https://github.com/joho/godotenv)
- [SQLite3 Driver](https://github.com/mattn/go-sqlite3)

---

