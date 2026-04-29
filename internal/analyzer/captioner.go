package analyzer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/genai"
)

type Captioner struct {
	client *genai.Client
}

func NewCaptioner(ctx context.Context, apiKey string) (*Captioner, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &Captioner{client: client}, nil
}

func (c *Captioner) GenerateSRT(ctx context.Context, videoPath string) (string, error) {
	log.Printf("Generating SRT for: %s", videoPath)

	// 1. Upload the file
	file, err := os.Open(videoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open video file: %v", err)
	}
	defer file.Close()

	fileName := filepath.Base(videoPath)
	uploadConfig := &genai.UploadFileConfig{
		DisplayName: fileName,
		MIMEType:    "video/mp4",
	}

	gFile, err := c.client.Files.Upload(ctx, file, uploadConfig)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %v", err)
	}
	defer func() {
		// Cleanup: delete the file from Gemini after processing
		_, err := c.client.Files.Delete(ctx, gFile.Name, nil)
		if err != nil {
			log.Printf("Warning: failed to delete file %s: %v", gFile.Name, err)
		}
	}()

	// 2. Wait for the file to be processed
	for gFile.State == "PROCESSING" {
		log.Printf("Waiting for file to be processed...")
		time.Sleep(5 * time.Second)
		gFile, err = c.client.Files.Get(ctx, gFile.Name, nil)
		if err != nil {
			return "", fmt.Errorf("failed to get file status: %v", err)
		}
	}

	if gFile.State == "FAILED" {
		return "", fmt.Errorf("file processing failed")
	}

	// 3. Generate SRT content
	prompt := `Generate a complete and accurate SubRip (.srt) subtitle file for this video. 
Translate all spoken dialogue to English.
Return ONLY the raw SRT content without any markdown formatting or extra text. 
Ensure the timestamps are precise and the text is natural English.`

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
		{FileData: &genai.FileData{FileURI: gFile.URI, MIMEType: gFile.MIMEType}},
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	// Use the model name from analyzer.go
	resp, err := c.client.Models.GenerateContent(ctx, "gemini-3-flash-preview", contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content returned from Gemini")
	}

	srtContent := resp.Candidates[0].Content.Parts[0].Text

	// Clean up markdown formatting if present
	srtContent = strings.TrimSpace(srtContent)
	if strings.HasPrefix(srtContent, "```") {
		lines := strings.Split(srtContent, "\n")
		if len(lines) > 2 {
			// Find the first line that looks like SRT content
			startIdx := 0
			for i, line := range lines {
				if strings.Contains(line, "-->") || (i > 0 && lines[i-1] == "1") {
					startIdx = i
					break
				}
			}
			// Find the last line that isn't ```
			endIdx := len(lines)
			for i := len(lines) - 1; i >= 0; i-- {
				if strings.Contains(lines[i], "```") {
					endIdx = i
					break
				}
			}
			if startIdx < endIdx {
				srtContent = strings.Join(lines[startIdx:endIdx], "\n")
			}
		}
	}

	return srtContent, nil
}
