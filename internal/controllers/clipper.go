package controllers

import (
	"errors"
	"fmt"
	"go-clipper/internal/analyzer"
	"go-clipper/internal/downloader"
	"go-clipper/internal/dtos"
	"go-clipper/internal/processor"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type ClipperController struct {
	analyzer  analyzer.Analyzer
	captioner analyzer.Captioner
}

func NewClipperController(analyzer analyzer.Analyzer, captioner analyzer.Captioner) *ClipperController {
	return &ClipperController{
		analyzer:  analyzer,
		captioner: captioner,
	}
}

func (cl *ClipperController) Create(c *gin.Context) {
	var request dtos.AnalyzeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !request.DownloadVideo && request.VideoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You need to specify video path if no download video process."})
		return
	}

	if _, err := os.Stat(request.OutputPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(request.OutputPath, 0755)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Directory created")
	}

	resAnalyze, err := cl.analyzer.AnalyzeVideoUrl(
		c.Request.Context(),
		request.YoutubeUrl,
		request.Count,
		request.MinimumDuration,
		request.MaximumDuration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.DownloadVideo {
		log.Println("Downloading video...")
		dl := downloader.NewDownloader(request.OutputPath, "")
		downloadedPath, err := dl.DownloadVideo(request.YoutubeUrl)
		if err != nil {
			log.Fatalf("Error downloading video: %v", err)
		}
		request.VideoPath = filepath.Clean(downloadedPath)
		fmt.Printf("Video downloaded to: %s\n", request.VideoPath)
	}

	proc := processor.NewProcessor(request.OutputPath, request.Ratio, false)
	for i, segment := range resAnalyze.Segments {
		if segment.Start == "" || segment.End == "" || segment.Hook == "" {
			fmt.Printf("[%d] Warning: Skipping invalid segment (missing start/end/hook): %+v\n", i+1, segment)
			continue
		}
		fmt.Printf("[%d] Processing: %s (%s to %s)\n", i+1, segment.Hook, segment.Start, segment.End)

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

		payload := dtos.CutClipPayload{
			SourceVideoPath: request.VideoPath,
			OutputPath:      request.OutputPath,
			Hook:            segment.Hook,
			StartSeconds:    segment.Start,
			EndSeconds:      segment.End,
		}
		clipPath, err := proc.CutVideoToClips(payload)
		if err != nil {
			fmt.Printf("Warning: Failed to cut clip '%s': %v\n", segment.Hook, err)
			continue
		}
		fmt.Printf("Clip saved: %s\n", clipPath)
	}

	c.JSON(http.StatusOK, gin.H{"data": resAnalyze})
}

func (cl *ClipperController) Analyze(c *gin.Context) {
	var request dtos.AnalyzeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resAnalyze, err := cl.analyzer.AnalyzeVideoUrl(
		c.Request.Context(),
		request.YoutubeUrl,
		request.Count,
		request.MaximumDuration,
		request.MaximumDuration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resAnalyze})
}

func (cl *ClipperController) GenerateCaption(c *gin.Context) {
	var request dtos.GenerateCaptionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clipsDir := "./output/clips"
	if request.ClipsPath != "" {
		clipsDir = request.ClipsPath
	}
	files, err := os.ReadDir(clipsDir)
	if err != nil {
		log.Fatalf("Error reading clips directory: %v", err)
	}

	srtPaths := make([]string, 0)

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
		srtContent, err := cl.captioner.GenerateSRT(c.Request.Context(), videoPath)
		if err != nil {
			fmt.Printf("Error generating SRT for %s: %v\n", file.Name(), err)
			continue
		}

		err = os.WriteFile(srtPath, []byte(srtContent), 0644)
		if err != nil {
			fmt.Printf("Error writing SRT file for %s: %v\n", file.Name(), err)
			continue
		}

		srtPaths = append(srtPaths, srtPath)

		fmt.Printf("Successfully generated SRT for %s\n", file.Name())
	}

	fmt.Println("\nAll captioning tasks completed!")

	c.JSON(http.StatusOK, gin.H{"data": srtPaths})
}
