package controllers

import (
	"errors"
	"fmt"
	"go-clipper/internal/analyzer"
	"go-clipper/internal/dtos"
	"go-clipper/internal/processor"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ClipperController struct {
	analyzer analyzer.Analyzer
}

func NewClipperController(analyzer analyzer.Analyzer) *ClipperController {
	return &ClipperController{
		analyzer: analyzer,
	}
}

func (cl *ClipperController) Create(c *gin.Context) {
	var request dtos.AnalyzeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		request.MaximumDuration,
		request.MaximumDuration)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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

		clipPath, err := proc.CutClip(request.VideoPath, segment.Start, segment.End, segment.Hook, subs)
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

func (cl *ClipperController) Process(c *gin.Context) {

}
