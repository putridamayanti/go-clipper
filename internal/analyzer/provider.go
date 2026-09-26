package analyzer

import (
	"context"
	"fmt"
	"go-clipper/internal/config"
	"go-clipper/internal/dtos"
)

// VideoAnalyzer finds clip segments and writes post descriptions for a video.
type VideoAnalyzer interface {
	AnalyzeVideoUrl(ctx context.Context, url string, payload dtos.AnalyzeRequest) (*dtos.AnalysisResult, error)
	GenerateDescription(ctx context.Context, url string) (*dtos.DescriptionResult, error)
}

// SubtitleGenerator writes English SRT subtitles for a local video file.
type SubtitleGenerator interface {
	GenerateSRT(ctx context.Context, videoPath string) (string, error)
}

// NewFromConfig builds the analyzer and captioner for the provider set by AI_PROVIDER.
func NewFromConfig(ctx context.Context, cfg *config.Config) (VideoAnalyzer, SubtitleGenerator, error) {
	switch cfg.Provider {
	case config.ProviderOpenAI:
		client := newOpenAIClient(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL)
		return &OpenAIAnalyzer{client: client, model: cfg.OpenAIModel, transcribeModel: cfg.OpenAITranscribeModel},
			&OpenAICaptioner{client: client, transcribeModel: cfg.OpenAITranscribeModel},
			nil
	case config.ProviderGemini:
		an, err := NewAnalyzer(ctx, cfg.GeminiAPIKey, cfg.GeminiModel)
		if err != nil {
			return nil, nil, fmt.Errorf("initializing Gemini analyzer: %v", err)
		}
		cp, err := NewCaptioner(ctx, cfg.GeminiAPIKeyForCaption, cfg.GeminiModelForCaption)
		if err != nil {
			return nil, nil, fmt.Errorf("initializing Gemini captioner: %v", err)
		}
		return an, cp, nil
	default:
		return nil, nil, fmt.Errorf("unknown AI provider %q", cfg.Provider)
	}
}
