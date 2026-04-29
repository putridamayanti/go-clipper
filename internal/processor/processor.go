package processor

import (
	"fmt"
	"go-clipper/internal/dtos"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Processor struct {
	OutputDir     string
	TargetRatio   string // e.g. "9:16", "1:1"
	BurnSubtitles bool
}

func NewProcessor(outputDir, targetRatio string, burnSubtitles bool) *Processor {
	return &Processor{
		OutputDir:     outputDir,
		TargetRatio:   targetRatio,
		BurnSubtitles: burnSubtitles,
	}
}

func (p *Processor) getDimensions() (int, int) {
	switch p.TargetRatio {
	case "9:16":
		return 1080, 1920
	case "1:1":
		return 1080, 1080
	case "4:5":
		return 1080, 1350
	case "16:9":
		return 1920, 1080
	default:
		return 0, 0 // No scaling
	}
}

type SubtitleLine struct {
	Start float64
	End   float64
	Text  string
}

func (p *Processor) CutVideoToClips(payload dtos.CutClipPayload) (string, error) {
	safeHook := strings.ReplaceAll(strings.ToLower(payload.Hook), " ", "_")
	safeHook = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, safeHook)

	log.Println("Cut video clip: ", safeHook)

	fileName := fmt.Sprintf("%s.mp4", safeHook)
	outputPath := filepath.Join(p.OutputDir, fileName)
	//if _, err := os.Stat(outputPath); os.IsNotExist(err) {
	//	err := os.Mkdir(p.OutputDir, 0755)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	fmt.Println("Directory created")
	//}

	var filters []string
	w, h := p.getDimensions()
	if w > 0 && h > 0 {
		// Scale to fit (contained) and pad to target ratio
		filters = append(filters, fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", w, h, w, h))
	}

	vf := strings.Join(filters, ",")

	args := []string{
		"-i", payload.SourceVideoPath,
		"-ss", payload.StartSeconds,
		"-to", payload.EndSeconds,
		"-avoid_negative_ts", "make_zero",
	}

	if vf != "" {
		args = append(args, "-vf", vf)
	}

	args = append(args,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "44100",
		"-map_metadata", "-1", // Strip metadata that might have old timestamps
		"-strict", "experimental",
		"-y",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)

	fmt.Printf("Clipping segment: %s -> %s (Hook: %s, Ratio: %s)\n",
		payload.StartSeconds, payload.EndSeconds, payload.Hook, p.TargetRatio)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to cut clip: %v\nOutput: %s", err, string(output))
	}

	return outputPath, nil
}

func (p *Processor) CutClip(videoPath, start, end, hook string, subtitles []SubtitleLine) (string, error) {
	// Clean hook for filename
	safeHook := strings.ReplaceAll(strings.ToLower(hook), " ", "_")
	safeHook = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, safeHook)

	fileName := fmt.Sprintf("%s.mp4", safeHook)
	outputPath := filepath.Join(p.OutputDir, fileName)

	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found in PATH")
	}

	// 1. Generate SRT file if subtitles exist
	var srtPath string
	if len(subtitles) > 0 {
		srtPath = filepath.Join(p.OutputDir, safeHook+".srt")
		err := p.generateSRT(srtPath, subtitles)
		if err != nil {
			return "", fmt.Errorf("failed to generate SRT: %v", err)
		}
		// We no longer remove the SRT file as the user wants to keep it
	}

	// 2. Determine filters
	var filters []string
	w, h := p.getDimensions()
	if w > 0 && h > 0 {
		// Scale to fit (contained) and pad to target ratio
		filters = append(filters, fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", w, h, w, h))
	}

	if srtPath != "" && p.BurnSubtitles {
		// Burn subtitles with some styling (centered, yellow text, outline)
		// Note: FFmpeg's subtitles filter path needs careful escaping on Windows, but on Mac/Linux it's usually fine.
		// We'll use absolute path for safety.
		absSRT, _ := filepath.Abs(srtPath)
		style := "FontSize=24,PrimaryColour=&H00FFFF,OutlineColour=&H000000,BorderStyle=1,Outline=1,Shadow=0,Alignment=2"
		filters = append(filters, fmt.Sprintf("subtitles='%s':force_style='%s'", absSRT, style))
	}

	vf := strings.Join(filters, ",")

	args := []string{
		"-i", videoPath,
		"-ss", start,
		"-to", end,
		"-avoid_negative_ts", "make_zero",
	}

	if vf != "" {
		args = append(args, "-vf", vf)
	}

	args = append(args,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a", "aac",
		"-b:a", "128k",
		"-ar", "44100",
		"-map_metadata", "-1", // Strip metadata that might have old timestamps
		"-strict", "experimental",
		"-y",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)

	fmt.Printf("Clipping segment: %s -> %s (Hook: %s, Ratio: %s, Subtitles: %t)\n", start, end, hook, p.TargetRatio, len(subtitles) > 0)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to cut clip: %v\nOutput: %s", err, string(output))
	}

	return outputPath, nil
}

func (p *Processor) generateSRT(path string, subtitles []SubtitleLine) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, line := range subtitles {
		start := formatSRTTime(line.Start)
		end := formatSRTTime(line.End)
		fmt.Fprintf(f, "%d\n%s --> %s\n%s\n\n", i+1, start, end, line.Text)
	}
	return nil
}

func formatSRTTime(seconds float64) string {
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	ms := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
