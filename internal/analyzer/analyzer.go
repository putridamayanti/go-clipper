package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type Segment struct {
	Start       string `json:"start"` // Format: HH:MM:SS or SS
	End         string `json:"end"`   // Format: HH:MM:SS or SS
	Description string `json:"description"`
	Hook        string `json:"hook"`
}

type AnalysisResult struct {
	Segments []Segment `json:"segments"`
}

type Analyzer struct {
	client *genai.Client
}

func NewAnalyzer(ctx context.Context, apiKey string) (*Analyzer, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &Analyzer{client: client}, nil
}

func (a *Analyzer) AnalyzeAudio(ctx context.Context, audioPath string) (*AnalysisResult, error) {
	// 1. Upload the audio file to Gemini File API
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %v", err)
	}
	defer file.Close()

	fmt.Println("Uploading audio to Gemini for analysis...")
	options := &genai.UploadFileOptions{DisplayName: "YouTube Audio"}
	gFile, err := a.client.UploadFile(ctx, "", file, options)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %v", err)
	}

	// 2. Generate content using the uploaded file
	model := a.client.GenerativeModel("gemini-3-flash-preview")

	// Set response MIME type to application/json for structured output
	model.ResponseMIMEType = "application/json"

	prompt := `
		Analyze the provided audio transcript. 
		1. Identify the most engaging and "viral" segments that would make great short-form clips (TikTok, Reels, Shorts).
		2. For each segment, provide the start and end timestamps (in seconds).
		3. Provide a brief description and a compelling "hook" title for each clip.
		
		Output the result in the following JSON format:
		{
			"segments": [
				{
					"start": "0",
					"end": "30",
					"description": "Explaining the core concept",
					"hook": "The secret no one tells you"
				}
			]
		}
	`

	resp, err := model.GenerateContent(ctx, genai.FileData{URI: gFile.URI}, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no analysis results returned")
	}

	// 3. Parse the JSON response
	var result AnalysisResult
	part := resp.Candidates[0].Content.Parts[0]
	if text, ok := part.(genai.Text); ok {
		err = json.Unmarshal([]byte(text), &result)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %v\nRaw: %s", err, string(text))
		}
	}

	// Cleanup: delete the uploaded file from Gemini
	_ = a.client.DeleteFile(ctx, gFile.Name)

	return &result, nil
}
