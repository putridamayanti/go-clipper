package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"go-clipper/internal/analyzer"
	"go-clipper/internal/config"
	"go-clipper/internal/downloader"
	"go-clipper/internal/processor"
)

func main() {
	youtubeURL := flag.String("url", "", "YouTube Video URL")
	inputPath := flag.String("input", "", "Local video file path (skips download)")
	//inputAudioPath := flag.String("audio", "", "Local audio file path (skips download)")
	outputDir := flag.String("out", "./output", "Directory to save clips")
	cookies := flag.String("cookies", "", "Extract cookies from browser (chrome, safari, firefox, etc.)")
	ratio := flag.String("ratio", "", "Target aspect ratio (9:16, 1:1, 4:5, 16:9)")
	count := flag.Int("count", 3, "Number of clips to generate")
	minLen := flag.Int("min", 15, "Minimum duration of each clip in seconds")
	burn := flag.Bool("burn", false, "Burn subtitles into video")
	flag.Parse()

	if *youtubeURL == "" && *inputPath == "" {
		fmt.Println("Usage: go run cmd/clipper/main.go -url <youtube_url> or -input <local_file> [-min 30] [-burn]")
		os.Exit(1)
	}

	// 1. Load Config
	cfg := config.LoadConfig()
	ctx := context.Background()

	// Ensure output directory exists
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// 2. Resolve Video Source
	var videoPath string
	//var audioPath string

	dl := downloader.NewDownloader(*outputDir, *cookies)

	if *inputPath != "" {
		absPath, err := filepath.Abs(*inputPath)
		if err != nil {
			log.Fatalf("Error resolving input path: %v", err)
		}
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			log.Fatalf("Input file does not exist: %s", absPath)
		}
		videoPath = absPath
		fmt.Printf("Using local video: %s\n", videoPath)

		// 3. Extract Audio
		//if *inputAudioPath == "" {
		//	path, err := dl.ExtractAudio(videoPath)
		//	if err != nil {
		//		log.Fatalf("Error extracting audio: %v", err)
		//	}
		//
		//	audioPath = path
		//} else {
		//	audioPath = *inputAudioPath
		//}
	} else {
		downloadedPath, err := dl.DownloadVideo(*youtubeURL)
		if err != nil {
			log.Fatalf("Error downloading video: %v", err)
		}
		videoPath = filepath.Clean(downloadedPath)
		fmt.Printf("Video downloaded to: %s\n", videoPath)
	}

	// 4. Analyze with Gemini
	az, err := analyzer.NewAnalyzer(ctx, cfg.GeminiAPIKey)
	if err != nil {
		log.Fatalf("Error initializing analyzer: %v", err)
	}

	//result, err := az.AnalyzeAudio(ctx, audioPath, *count, *minLen)
	//if err != nil {
	//	log.Fatalf("Error analyzing audio: %v", err)
	//}

	result, err := az.AnalyzeVideoUrl(ctx, *youtubeURL, *count, *minLen, 60)
	if err != nil {
		log.Fatalf("Error analyzing video url: %v", err)
	}

	fmt.Printf("Found %d viral segments!\n", len(result.Segments))

	// 5. Cut Clips
	proc := processor.NewProcessor(*outputDir, *ratio, *burn)
	for i, segment := range result.Segments {
		if segment.Start == "" || segment.End == "" || segment.Hook == "" {
			fmt.Printf("[%d] Warning: Skipping invalid segment (missing start/end/hook): %+v\n", i+1, segment)
			continue
		}
		fmt.Printf("[%d] Processing: %s (%s to %s)\n", i+1, segment.Hook, segment.Start, segment.End)

		// Convert analyzer.SubtitleLine to processor.SubtitleLine
		// Handle potential absolute timestamps from Gemini (relative to original video vs segment start)
		segmentStart, _ := strconv.ParseFloat(segment.Start, 64)
		var subs []processor.SubtitleLine
		for _, s := range segment.Subtitles {
			startTs := s.Start
			endTs := s.End

			// If the subtitle start is >= segment start, it's likely an absolute timestamp.
			// We need it to be relative to 0 for the clipped video.
			if startTs >= segmentStart {
				startTs -= segmentStart
				endTs -= segmentStart
			}

			subs = append(subs, processor.SubtitleLine{
				Start: startTs,
				End:   endTs,
				Text:  s.Text,
			})
		}

		clipPath, err := proc.CutClip(videoPath, segment.Start, segment.End, segment.Hook, subs)
		if err != nil {
			fmt.Printf("Warning: Failed to cut clip '%s': %v\n", segment.Hook, err)
			continue
		}
		fmt.Printf("Clip saved: %s\n", clipPath)
	}

	fmt.Println("\nAll done! Check the output directory for your viral clips.")
}
