package models

type CryptoNewsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Body        string `json:"body"`
		PublishedOn int64  `json:"published_on"`
		URL         string `json:"url"`
	} `json:"Data"`
}

type TradingViewAlert struct {
	StrategyName string  `json:"strategyName"`
	Ticker       string  `json:"ticker"`
	Price        float64 `json:"price"`
	Message      string  `json:"message"`
}
