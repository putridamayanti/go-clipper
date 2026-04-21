package processor

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Processor struct {
	OutputDir   string
	TargetRatio string // e.g. "9:16", "1:1"
}

func NewProcessor(outputDir, targetRatio string) *Processor {
	return &Processor{
		OutputDir:   outputDir,
		TargetRatio: targetRatio,
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

func (p *Processor) CutClip(videoPath, start, end, hook string) (string, error) {
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

	// Determine filters
	vf := ""
	w, h := p.getDimensions()
	if w > 0 && h > 0 {
		// Scale to fit (contained) and pad to target ratio
		vf = fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2", w, h, w, h)
	}

	args := []string{
		"-ss", start,
		"-to", end,
		"-i", videoPath,
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
		"-strict", "experimental",
		"-y",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)

	fmt.Printf("Clipping segment: %s -> %s (Hook: %s, Ratio: %s)\n", start, end, hook, p.TargetRatio)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to cut clip: %v\nOutput: %s", err, string(output))
	}

	return outputPath, nil
}
