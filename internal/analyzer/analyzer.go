package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"go-clipper/internal/dtos"
	"log"
	"strings"
	"time"

	"google.golang.org/genai"
	//"github.com/google/generative-ai-go/genai"
)

type SubtitleLine struct {
	Start float64 `json:"start"` // Seconds relative to segment start
	End   float64 `json:"end"`   // Seconds relative to segment start
	Text  string  `json:"text"`
}

type Analyzer struct {
	client *genai.Client
	model  string
}

func NewAnalyzer(ctx context.Context, apiKey, model string) (*Analyzer, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &Analyzer{client: client, model: model}, nil
}

func (a *Analyzer) AnalyzeVideoUrl(ctx context.Context, url string, payload dtos.AnalyzeRequest) (*dtos.AnalysisResult, error) {
	start := time.Now()

	log.Printf("Analyzing video url: %s", url)
	prompt := analyzePrompt(payload, "Analyze the provided video.")

	// Clean YouTube URL to remove extra parameters
	cleanURL := url
	if strings.Contains(cleanURL, "watch?v=") {
		parts := strings.Split(cleanURL, "&")
		cleanURL = parts[0]
	}

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
		genai.NewPartFromURI(cleanURL, "video/mp4"),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"description": {Type: genai.TypeString},
			"segments": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"start":       {Type: genai.TypeString},
						"end":         {Type: genai.TypeString},
						"description": {Type: genai.TypeString},
						"hook":        {Type: genai.TypeString},
					},
					Required: []string{"start", "end", "description", "hook"},
				},
			},
		},
		Required: []string{"description", "segments"},
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}

	resp, err := a.client.Models.GenerateContent(
		ctx,
		a.model,
		contents,
		config,
	)

	latency := time.Since(start)
	log.Printf("Analyzing video %s latency: %s", url, latency)

	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no analysis results returned")
	}

	cand := resp.Candidates[0]
	log.Printf("Finish Reason: %v", cand.FinishReason)

	if len(cand.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response")
	}

	// 3. Parse the JSON response
	var result dtos.AnalysisResult
	rawText := cand.Content.Parts[0].Text
	if rawText != "" {
		log.Printf("Raw Analysis Output: %s", rawText)

		err = json.Unmarshal([]byte(rawText), &result)
		if err != nil {
			// Fallback: try to strip markdown if the model still includes it
			cleanText := rawText
			if strings.Contains(cleanText, "```json") {
				parts := strings.Split(cleanText, "```json")
				if len(parts) > 1 {
					cleanText = strings.Split(parts[1], "```")[0]
				}
			} else if strings.HasPrefix(cleanText, "```") {
				cleanText = strings.TrimPrefix(cleanText, "```")
				cleanText = strings.TrimSuffix(cleanText, "```")
			}
			cleanText = strings.TrimSpace(cleanText)

			err = json.Unmarshal([]byte(cleanText), &result)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %v\nRaw: %s", err, rawText)
			}
		}
	}

	log.Println("Segments: ", len(result.Segments))

	return &result, nil
}

func (a *Analyzer) GenerateDescription(ctx context.Context, url string) (*dtos.DescriptionResult, error) {
	start := time.Now()

	prompt := descriptionPrompt(fmt.Sprintf("Create caption/description for this youtube video %s.", url))

	cleanURL := url
	if strings.Contains(cleanURL, "watch?v=") {
		parts := strings.Split(cleanURL, "&")
		cleanURL = parts[0]
	}

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
		genai.NewPartFromURI(cleanURL, "video/mp4"),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	schema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"description": {Type: genai.TypeString},
			"tags":        {Type: genai.TypeString},
		},
		Required: []string{"description", "tags"},
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}

	resp, err := a.client.Models.GenerateContent(
		ctx,
		a.model,
		contents,
		config,
	)

	latency := time.Since(start)
	log.Printf("Analyzing video %s latency: %s", url, latency)

	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no analysis results returned")
	}

	cand := resp.Candidates[0]
	log.Printf("Finish Reason: %v", cand.FinishReason)

	if len(cand.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response")
	}

	var result dtos.DescriptionResult
	rawText := cand.Content.Parts[0].Text
	if rawText != "" {
		log.Printf("Raw Analysis Output: %s", rawText)

		err = json.Unmarshal([]byte(rawText), &result)
		if err != nil {
			// Fallback: try to strip markdown if the model still includes it
			cleanText := rawText
			if strings.Contains(cleanText, "```json") {
				parts := strings.Split(cleanText, "```json")
				if len(parts) > 1 {
					cleanText = strings.Split(parts[1], "```")[0]
				}
			} else if strings.HasPrefix(cleanText, "```") {
				cleanText = strings.TrimPrefix(cleanText, "```")
				cleanText = strings.TrimSuffix(cleanText, "```")
			}
			cleanText = strings.TrimSpace(cleanText)

			err = json.Unmarshal([]byte(cleanText), &result)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %v\nRaw: %s", err, rawText)
			}
		}
	}

	return &result, nil
}
