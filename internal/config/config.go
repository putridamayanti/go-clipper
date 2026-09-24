package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GeminiAPIKey           string
	GeminiAPIKeyForCaption string
	GeminiModel            string
	GeminiModelForCaption  string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY is required")
	}

	apiKeyForCaption := os.Getenv("GEMINI_API_KEY_FOR_CAPTION")
	if apiKeyForCaption == "" {
		log.Fatal("GEMINI_API_KEY_FOR_CAPTION is required")
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-3-flash-preview"
	}

	modelForCaption := os.Getenv("GEMINI_MODEL_FOR_CAPTION")
	if modelForCaption == "" {
		modelForCaption = model
	}

	return &Config{
		GeminiAPIKey:           apiKey,
		GeminiAPIKeyForCaption: apiKeyForCaption,
		GeminiModel:            model,
		GeminiModelForCaption:  modelForCaption,
	}
}
