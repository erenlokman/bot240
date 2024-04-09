package utils

import (
	"encoding/json"
	"log"

	"github.com/erenlokman/bot240/config"
	"github.com/go-resty/resty/v2"
)

func GetOpenAIResponse(prompt string) string {
	client := resty.New()
	apiKey := config.GetEnvVar("OPENAI_API_KEY")

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
