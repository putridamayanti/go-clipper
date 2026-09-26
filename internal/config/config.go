package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

const (
	ProviderGemini = "gemini"
	ProviderOpenAI = "openai"
)

type Config struct {
	// AI_PROVIDER: "gemini" (default) or "openai"
	Provider string

	GeminiAPIKey           string
	GeminiAPIKeyForCaption string
	GeminiModel            string
	GeminiModelForCaption  string // Defaults to GeminiModel

	OpenAIAPIKey         string
	OpenAIBaseURL         string
	OpenAIModel           string // Chat model used to analyze segments and write descriptions
	OpenAITranscribeModel string // Speech-to-text model; must support SRT output (e.g. whisper-1)
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment variables")
	}

	cfg := &Config{
		Provider:               strings.ToLower(getEnv("AI_PROVIDER", ProviderGemini)),
		GeminiAPIKey:           os.Getenv("GEMINI_API_KEY"),
		GeminiAPIKeyForCaption: os.Getenv("GEMINI_API_KEY_FOR_CAPTION"),
		GeminiModel:            getEnv("GEMINI_MODEL", "gemini-3-flash-preview"),
		GeminiModelForCaption:  os.Getenv("GEMINI_MODEL_FOR_CAPTION"),
		OpenAIAPIKey:           os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:          strings.TrimRight(getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"), "/"),
		OpenAIModel:            os.Getenv("OPENAI_MODEL"),
		OpenAITranscribeModel:  getEnv("OPENAI_TRANSCRIBE_MODEL", "whisper-1"),
	}
	if cfg.GeminiModelForCaption == "" {
		cfg.GeminiModelForCaption = cfg.GeminiModel
	}

	switch cfg.Provider {
	case ProviderGemini:
		if cfg.GeminiAPIKey == "" {
			log.Fatal("GEMINI_API_KEY is required")
		}
		if cfg.GeminiAPIKeyForCaption == "" {
			log.Fatal("GEMINI_API_KEY_FOR_CAPTION is required")
		}
	case ProviderOpenAI:
		if cfg.OpenAIAPIKey == "" {
			log.Fatal("OPENAI_API_KEY is required when AI_PROVIDER=openai")
		}
		if cfg.OpenAIModel == "" {
			log.Fatal("OPENAI_MODEL is required when AI_PROVIDER=openai")
		}
	default:
		log.Fatalf("Unknown AI_PROVIDER %q (use %q or %q)", cfg.Provider, ProviderGemini, ProviderOpenAI)
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
