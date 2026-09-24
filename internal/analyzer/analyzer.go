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

	if payload.StartTimestamp == "" {
		payload.StartTimestamp = "start"
	}

	if payload.EndTimestamp == "" {
		payload.EndTimestamp = "end"
	}

	log.Printf("Analyzing video url: %s", url)
	prompt := fmt.Sprintf(`
		Analyze the provided video.
		1. Identify exactly %d of the most engaging and "viral" segments that would make great short-form clips (TikTok, Reels, Shorts).
		2. Ensure each segment is at least %d seconds long and maximum %d seconds.
		3. For each segment, provide the start and end timestamps (in seconds). THESE ARE MANDATORY.
		4. Provide a brief description and a compelling "hook" title for each clip. THE HOOK MUST NOT BE EMPTY.
		5. Analyze only from duration %s to %s

		Output the result as a single JSON object in the following format:
		{
			"description": "Short summary of the video",
			"segments": [
				{
					"start": "30",
					"end": "60",
					"description": "Deep insight about AI",
					"hook": "The Truth About AI"
				}
			]
		}
	`, payload.Count, payload.MinimumDuration, payload.MaximumDuration, payload.StartTimestamp, payload.EndTimestamp)

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

	prompt := fmt.Sprintf(`
		Create caption/description for this youtube video %s.
		1. Include the tags to separate by commas without # with At least 400 characters. MANDATORY. Make sure at leas 400 characters. Make sure it will help content viral.
		2. No timestamp/seconds in the description.
		3. Will repost this in youtube short and instagram

		Output the result as a single JSON object in the following format:
		{
			"description": "The chaos continues in this TRIP",
			"tags": "viral video, funny, fun"
		}
	`, url)

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
