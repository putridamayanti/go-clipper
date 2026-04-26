package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GeminiAPIKey           string
	GeminiAPIKeyForCaption string
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

	return &Config{
		GeminiAPIKey:           apiKey,
		GeminiAPIKeyForCaption: apiKeyForCaption,
	}
}
