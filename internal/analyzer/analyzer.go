package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"
	//"github.com/google/generative-ai-go/genai"
)

type SubtitleLine struct {
	Start float64 `json:"start"` // Seconds relative to segment start
	End   float64 `json:"end"`   // Seconds relative to segment start
	Text  string  `json:"text"`
}

type Segment struct {
	Start       string         `json:"start"` // Format: SS
	End         string         `json:"end"`   // Format: SS
	Description string         `json:"description"`
	Hook        string         `json:"hook"`
	Subtitles   []SubtitleLine `json:"subtitles"`
}

type AnalysisResult struct {
	Description string    `json:"description"`
	Segments    []Segment `json:"segments"`
}

type Analyzer struct {
	client *genai.Client
}

func NewAnalyzer(ctx context.Context, apiKey string) (*Analyzer, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &Analyzer{client: client}, nil
}

func (a *Analyzer) AnalyzeVideoUrl(ctx context.Context, url string, count, minSeconds, maxSeconds int) (*AnalysisResult, error) {
	log.Printf("Analyzing video url: %s", url)
	prompt := fmt.Sprintf(`
		Analyze the provided video.
		1. Identify exactly %d of the most engaging and "viral" segments that would make great short-form clips (TikTok, Reels, Shorts).
		2. Ensure each segment is at least %d seconds long and maximum %d seconds.
		3. For each segment, provide the start and end timestamps (in seconds). THESE ARE MANDATORY.
		4. For each segment, provide a word-for-word (or sentence-by-sentence) English translation transcript with timestamps relative to the start of that segment.
		5. Provide a brief description and a compelling "hook" title for each clip. THE HOOK MUST NOT BE EMPTY.

		Output the result as a single JSON object in the following format:
		{
			"description": "Short summary of the video",
			"segments": [
				{
					"start": "30",
					"end": "60",
					"description": "Deep insight about AI",
					"hook": "The Truth About AI",
					"subtitles": [
						{"start": 0.5, "end": 2.0, "text": "AI is changing the world"},
						{"start": 2.5, "end": 4.5, "text": "And we need to be ready"}
					]
				}
			]
		}
	`, count, minSeconds, maxSeconds)

	// Clean YouTube URL to remove extra parameters
	cleanURL := url
	if strings.Contains(cleanURL, "watch?v=") {
		parts := strings.Split(cleanURL, "&")
		cleanURL = parts[0]
	}

	// parts := []*genai.Part{
	// 	genai.NewPartFromText(prompt),
	// 	genai.NewPartFromURI(cleanURL, "video/mp4"),
	// }

	// config := &genai.GenerateContentConfig{
	// 	ResponseMIMEType: "application/json",
	// }
	// resp, err := a.client.Models.GenerateContent(ctx, "gemini-3-flash-preview", parts, config)

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
		genai.NewPartFromURI(cleanURL, "video/mp4"),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	}

	resp, err := a.client.Models.GenerateContent(
		ctx,
		"gemini-3-flash-preview",
		contents,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no analysis results returned")
	}

	// 3. Parse the JSON response
	var result AnalysisResult
	part := resp.Candidates[0].Content.Parts[0]
	if part.Text != "" {
		log.Println(part.Text)

		// Clean the response from potential markdown code blocks
		rawText := part.Text
		if strings.HasPrefix(rawText, "```json") {
			rawText = strings.TrimPrefix(rawText, "```json")
			rawText = strings.TrimSuffix(rawText, "```")
		} else if strings.HasPrefix(rawText, "```") {
			rawText = strings.TrimPrefix(rawText, "```")
			rawText = strings.TrimSuffix(rawText, "```")
		}
		rawText = strings.TrimSpace(rawText)

		// Handle both object and array responses
		if strings.HasPrefix(rawText, "[") {
			var segments []Segment
			err = json.Unmarshal([]byte(rawText), &segments)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON array: %v\nRaw: %s", err, rawText)
			}
			result.Segments = segments
			result.Description = "Generated segments"
		} else {
			err = json.Unmarshal([]byte(rawText), &result)
			if err != nil {
				return nil, fmt.Errorf("failed to parse JSON object: %v\nRaw: %s", err, rawText)
			}
		}
	}

	return &result, nil
}

//func (a *Analyzer) AnalyzeAudio(ctx context.Context, audioPath string, count, minSeconds int) (*AnalysisResult, error) {
//	// 1. Upload the audio file to Gemini File API
//	file, err := os.Open(audioPath)
//	if err != nil {
//		return nil, fmt.Errorf("failed to open audio file: %v", err)
//	}
//	defer file.Close()
//
//	fmt.Println("Uploading audio to Gemini for analysis...")
//	options := &genai.UploadFileOptions{DisplayName: "YouTube Audio"}
//	gFile, err := a.client.UploadFile(ctx, "", file, options)
//	if err != nil {
//		return nil, fmt.Errorf("failed to upload file: %v", err)
//	}
//
//	// 2. Generate content using the uploaded file
//	model := a.client.GenerativeModel("gemini-3-flash-preview")
//
//	// Set response MIME type to application/json for structured output
//	model.ResponseMIMEType = "application/json"
//
//	prompt := fmt.Sprintf(`
//		Analyze the provided audio transcript.
//		1. Identify exactly %d of the most engaging and "viral" segments that would make great short-form clips (TikTok, Reels, Shorts).
//		2. Ensure each segment is at least %d seconds long.
//		3. For each segment, provide the start and end timestamps (in seconds).
//		4. Provide a brief description and a compelling "hook" title for each clip.
//
//		Output the result in the following JSON format:
//		{
//			"segments": [
//				{
//					"start": "0",
//					"end": "30",
//					"description": "Explaining the core concept",
//					"hook": "The secret no one tells you"
//				}
//			]
//		}
//	`, count, minSeconds)
//
//	resp, err := model.GenerateContent(ctx, genai.FileData{URI: gFile.URI}, genai.Text(prompt))
//	if err != nil {
//		return nil, fmt.Errorf("failed to generate content: %v", err)
//	}
//
//	if len(resp.Candidates) == 0 {
//		return nil, fmt.Errorf("no analysis results returned")
//	}
//
//	// 3. Parse the JSON response
//	var result AnalysisResult
//	part := resp.Candidates[0].Content.Parts[0]
//	if text, ok := part.(genai.Text); ok {
//		err = json.Unmarshal([]byte(text), &result)
//		if err != nil {
//			return nil, fmt.Errorf("failed to parse JSON response: %v\nRaw: %s", err, string(text))
//		}
//	}
//
//	// Cleanup: delete the uploaded file from Gemini
//	_ = a.client.DeleteFile(ctx, gFile.Name)
//
//	return &result, nil
//}
