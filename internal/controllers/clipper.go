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
	"strings"

	"github.com/gin-gonic/gin"
)

type ClipperController struct {
	analyzer  analyzer.VideoAnalyzer
	captioner analyzer.SubtitleGenerator
}

func NewClipperController(analyzer analyzer.VideoAnalyzer, captioner analyzer.SubtitleGenerator) *ClipperController {
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

	outputPath := "./output"
	if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(outputPath, 0755)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Directory created")
	}

	// Download first so the analyzer can use the local video and its subtitles
	// (the OpenAI provider needs them; Gemini reads the YouTube URL directly)
	if request.DownloadVideo {
		log.Println("Downloading video...")
		dl := downloader.NewDownloader(outputPath, "")
		downloadedPath, err := dl.DownloadVideo(request.YoutubeUrl, request.IncludeSubtitle)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		request.VideoPath = filepath.Clean(downloadedPath)
		fmt.Printf("Video downloaded to: %s\n", request.VideoPath)
	}

	var resAnalyze *dtos.AnalysisResult

	if request.WithoutAnalyze && len(request.Segments) > 0 {
		resAnalyze = &dtos.AnalysisResult{
			Description: "",
			Segments:    request.Segments,
		}
	} else {
		result, err := cl.analyzer.AnalyzeVideoUrl(
			c.Request.Context(),
			request.YoutubeUrl,
			request)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resAnalyze = result
	}

	if resAnalyze == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You need to specify analyze video url"})
		return
	}

	// Use the YouTube subtitle file downloaded next to the source video, if any
	sourceSRT := downloader.FindSubtitleFile(request.VideoPath)
	if sourceSRT != "" {
		fmt.Printf("Using subtitles: %s\n", sourceSRT)
	}

	proc := processor.NewProcessor(request.OutputPath, request.Ratio, false)
	clips := make([]dtos.ClipResult, 0)
	for i, segment := range resAnalyze.Segments {
		if segment.Start == "" || segment.End == "" || segment.Hook == "" {
			fmt.Printf("[%d] Warning: Skipping invalid segment (missing start/end/hook): %+v\n", i+1, segment)
			continue
		}
		fmt.Printf("[%d] Processing: %s (%s to %s)\n", i+1, segment.Hook, segment.Start, segment.End)

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
		clip := dtos.ClipResult{ClipPath: clipPath}

		if sourceSRT != "" {
			srtPath, err := proc.CutSubtitle(sourceSRT, clipPath, segment.Start, segment.End)
			if err != nil {
				fmt.Printf("Warning: Failed to cut subtitle for '%s': %v\n", segment.Hook, err)
			} else if srtPath != "" {
				fmt.Printf("Subtitle saved: %s\n", srtPath)
				clip.SrtPath = srtPath
				clip.SubtitleSource = "youtube"
			}
		} else if request.WithCaption {
			// Fall back to AI captions only when YouTube has no subtitles
			srtPath, err := cl.generateClipSRT(c, clipPath)
			if err != nil {
				fmt.Printf("Warning: Failed to generate AI subtitle for '%s': %v\n", segment.Hook, err)
			} else {
				fmt.Printf("AI subtitle saved: %s\n", srtPath)
				clip.SrtPath = srtPath
				clip.SubtitleSource = "ai"
			}
		}

		clips = append(clips, clip)
	}

	c.JSON(http.StatusOK, gin.H{"data": resAnalyze, "clips": clips})
}

func (cl *ClipperController) generateClipSRT(c *gin.Context, clipPath string) (string, error) {
	srtContent, err := cl.captioner.GenerateSRT(c.Request.Context(), clipPath)
	if err != nil {
		return "", err
	}
	srtPath := strings.TrimSuffix(clipPath, ".mp4") + ".srt"
	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		return "", err
	}
	return srtPath, nil
}

func (cl *ClipperController) Download(c *gin.Context) {
	var request dtos.DownloadVideoRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	outputPath := "./output"
	if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(outputPath, 0755)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Directory created")
	}

	log.Println("Downloading video...")
	dl := downloader.NewDownloader(outputPath, "")
	downloadedPath, err := dl.DownloadVideo(request.YoutubeUrl, request.IncludeSubtitle)
	if err != nil {
		log.Fatalf("Error downloading video: %v", err)
	}
	request.VideoPath = filepath.Clean(downloadedPath)
	fmt.Printf("Video downloaded to: %s\n", request.VideoPath)

	c.JSON(http.StatusOK, gin.H{
		"data":          request.VideoPath,
		"subtitle_path": downloader.FindSubtitleFile(request.VideoPath),
	})
}

func (cl *ClipperController) DownloadInstagram(c *gin.Context) {
	var request dtos.DownloadInstagramRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !downloader.IsInstagramURL(request.Url) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url must be an Instagram post link"})
		return
	}

	outputPath := "./output"
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dl := downloader.NewDownloader(outputPath, request.CookiesBrowser)
	paths, err := dl.DownloadInstagramImages(request.Url, request.IncludeVideos)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Printf("Downloaded %d Instagram file(s)\n", len(paths))

	c.JSON(http.StatusOK, gin.H{"data": paths})
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
		request)
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

func (cl *ClipperController) GenerateDescription(c *gin.Context) {
	var request dtos.AnalyzeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := cl.analyzer.GenerateDescription(c.Request.Context(), request.YoutubeUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}
