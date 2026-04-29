package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go-clipper/internal/analyzer"
	"go-clipper/internal/config"
)

func main() {
	// 1. Load Config
	cfg := config.LoadConfig()
	ctx := context.Background()

	// 2. Initialize Captioner
	cp, err := analyzer.NewCaptioner(ctx, cfg.GeminiAPIKeyForCaption)
	if err != nil {
		log.Fatalf("Error initializing captioner: %v", err)
	}

	clipsDir := "./output/clips"
	files, err := os.ReadDir(clipsDir)
	if err != nil {
		log.Fatalf("Error reading clips directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".mp4") {
			continue
		}

		videoPath := filepath.Join(clipsDir, file.Name())
		srtPath := strings.TrimSuffix(videoPath, ".mp4") + ".srt"

		// Check if SRT already exists
		if _, err := os.Stat(srtPath); err == nil {
			fmt.Printf("SRT already exists for %s, skipping...\n", file.Name())
			continue
		}

		fmt.Printf("Processing %s...\n", file.Name())
		srtContent, err := cp.GenerateSRT(ctx, videoPath)
		if err != nil {
			fmt.Printf("Error generating SRT for %s: %v\n", file.Name(), err)
			continue
		}

		err = os.WriteFile(srtPath, []byte(srtContent), 0644)
		if err != nil {
			fmt.Printf("Error writing SRT file for %s: %v\n", file.Name(), err)
			continue
		}

		fmt.Printf("Successfully generated SRT for %s\n", file.Name())
	}

	fmt.Println("\nAll captioning tasks completed!")
}
