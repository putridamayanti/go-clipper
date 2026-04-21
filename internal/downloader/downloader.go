package downloader

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Downloader struct {
	OutputDir      string
	CookiesBrowser string // e.g. "chrome", "safari", "firefox"
}

func NewDownloader(outputDir, cookiesBrowser string) *Downloader {
	return &Downloader{
		OutputDir:      outputDir,
		CookiesBrowser: cookiesBrowser,
	}
}

func (d *Downloader) DownloadVideo(url string) (string, error) {
	outputTemplate := filepath.Join(d.OutputDir, "%(title)s.%(ext)s")

	// Check if yt-dlp is available
	_, err := exec.LookPath("yt-dlp")
	if err != nil {
		return "", fmt.Errorf("yt-dlp not found in PATH. Please install it with 'brew install yt-dlp'")
	}

	// Download video + audio
	// Use --no-warnings to keep stdout clean for path extraction
	args := []string{
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
		"--merge-output-format", "mp4",
		"-o", outputTemplate,
		"--print", "after_move:filepath",
		"--no-warnings",
	}

	if d.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", d.CookiesBrowser)
	}

	args = append(args, url)

	cmd := exec.Command("yt-dlp", args...)

	fmt.Printf("Downloading video: %s\n", url)

	// Separate stdout to get only the printed filepath
	stdout, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to download video: %v", err)
	}

	// Clean up path (remove newlines/spaces)
	videoPath := strings.TrimSpace(string(stdout))
	if videoPath == "" {
		return "", fmt.Errorf("could not determine downloaded video path")
	}

	return videoPath, nil
}

func (d *Downloader) ExtractAudio(videoPath string) (string, error) {
	audioPath := videoPath[:len(videoPath)-len(filepath.Ext(videoPath))] + ".mp3"

	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found in PATH. Please install it with 'brew install ffmpeg'")
	}

	cmd := exec.Command("ffmpeg", "-i", videoPath, "-vn", "-acodec", "libmp3lame", "-y", audioPath)
	fmt.Printf("Extracting audio to: %s\n", audioPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to extract audio: %v\nOutput: %s", err, string(output))
	}

	return audioPath, nil
}
