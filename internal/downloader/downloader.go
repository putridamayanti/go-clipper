package downloader

import (
	"bytes"
	"fmt"
	"log"
	"os"
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

func (d *Downloader) DownloadVideo(url string, includeSubtitle bool) (string, error) {
	// Use ToSlash to ensure forward slashes in the template, which yt-dlp handles better across platforms
	outputTemplate := filepath.ToSlash(filepath.Join(d.OutputDir, "%(title)s.%(ext)s"))

	// Check if yt-dlp is available
	_, err := exec.LookPath("yt-dlp")
	if err != nil {
		return "", fmt.Errorf("yt-dlp not found in PATH. Please install it")
	}

	// Check if ffmpeg is available (required for merging)
	_, err = exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found in PATH. yt-dlp requires ffmpeg to merge video and audio")
	}

	// Download video + audio
	// Use --no-warnings to keep stdout clean for path extraction
	args := []string{
		"-f", "bestvideo+bestaudio/best",
		"--merge-output-format", "mp4",
		"-o", outputTemplate,
		"--print", "after_move:filepath",
		"--no-warnings",
	}

	if includeSubtitle {
		args = append(args,
			"--ignore-errors",
			"--no-abort-on-error",
			"--write-subs",
			"--sub-langs", "en",
			"--convert-subs", "srt",
		)
	}

	args = append(args, d.commonArgs()...)

	log.Println(args)

	args = append(args, url)

	cmd := exec.Command("yt-dlp", args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	fmt.Printf("Downloading video: %s\n", url)

	err = cmd.Run()

	stdout := stdoutBuf.String()
	log.Println(stdout)

	// Clean up path (remove newlines/spaces)
	// We take the last line because yt-dlp might output other info even with --no-warnings
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		videoPath := strings.TrimSpace(lines[len(lines)-1])
		if _, statErr := os.Stat(videoPath); statErr == nil {
			// Subtitles are kept as a sidecar .srt next to the video so each clip
			// can get its own sliced .srt later (see FindSubtitleFile).
			if includeSubtitle {
				if srtPath := FindSubtitleFile(videoPath); srtPath != "" {
					fmt.Printf("YouTube subtitles saved to: %s\n", srtPath)
				} else {
					fmt.Println("Warning: This video has no English subtitles on YouTube.")
				}
			}
			return videoPath, nil
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to download video: %v\nStderr: %s", err, stderrBuf.String())
	}

	return "", fmt.Errorf("could not determine downloaded video path. Output: %s", stdout)
}

// commonArgs returns the yt-dlp flags shared by every download.
func (d *Downloader) commonArgs() []string {
	var args []string

	// yt-dlp needs a JS runtime to get full YouTube formats and only enables deno
	// by default, so fall back to node when deno isn't installed.
	if _, err := exec.LookPath("deno"); err != nil {
		if _, err := exec.LookPath("node"); err == nil {
			args = append(args, "--js-runtimes", "node")
		}
	}

	if d.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", d.CookiesBrowser)
	}
	return args
}

// DownloadTranscriptSource fetches what's needed to build a transcript without
// downloading the video: English subtitles (uploaded or auto-generated) if the
// video has them, otherwise the audio track. Files go into a temp directory
// that the returned cleanup func removes. Exactly one of srtPath/audioPath is set.
func (d *Downloader) DownloadTranscriptSource(url string) (srtPath, audioPath string, cleanup func(), err error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return "", "", nil, fmt.Errorf("yt-dlp not found in PATH. Please install it")
	}

	tmpDir, err := os.MkdirTemp("", "go-clipper-*")
	if err != nil {
		return "", "", nil, err
	}
	cleanup = func() { _ = os.RemoveAll(tmpDir) }

	// 1. Subtitles only. Errors are ignored: a video without subtitles falls back to audio.
	subArgs := []string{
		"--skip-download",
		"--write-subs",
		"--write-auto-subs",
		"--sub-langs", "en.*",
		"--convert-subs", "srt",
		"--ignore-errors",
		"--no-warnings",
		"-o", filepath.ToSlash(filepath.Join(tmpDir, "subs.%(ext)s")),
	}
	subArgs = append(subArgs, d.commonArgs()...)
	fmt.Printf("Fetching subtitles for transcript: %s\n", url)
	_ = exec.Command("yt-dlp", append(subArgs, url)...).Run()

	if matches, _ := filepath.Glob(filepath.Join(tmpDir, "*.srt")); len(matches) > 0 {
		return matches[0], "", cleanup, nil
	}

	// 2. Audio only
	audioArgs := []string{
		"-f", "bestaudio/best",
		"--no-warnings",
		"-o", filepath.ToSlash(filepath.Join(tmpDir, "audio.%(ext)s")),
		"--print", "after_move:filepath",
	}
	audioArgs = append(audioArgs, d.commonArgs()...)
	fmt.Printf("No subtitles found, downloading audio for transcript: %s\n", url)

	cmd := exec.Command("yt-dlp", append(audioArgs, url)...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("failed to download audio: %v\nStderr: %s", err, stderrBuf.String())
	}

	lines := strings.Split(strings.TrimSpace(stdoutBuf.String()), "\n")
	audioPath = strings.TrimSpace(lines[len(lines)-1])
	if _, err := os.Stat(audioPath); err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("could not find downloaded audio. Output: %s", stdoutBuf.String())
	}
	return "", audioPath, cleanup, nil
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

// FindSubtitleFile returns the .srt that sits next to videoPath
// (e.g. "Title.srt" or yt-dlp's "Title.en.srt"), or "" if there is none.
func FindSubtitleFile(videoPath string) string {
	dir := filepath.Dir(videoPath)
	ext := filepath.Ext(videoPath)
	baseWithoutExt := filepath.Base(videoPath[:len(videoPath)-len(ext)])

	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		// Require the "." after the base name so "Title" doesn't match "Title Part 2.en.srt"
		if strings.HasPrefix(name, baseWithoutExt+".") && strings.HasSuffix(name, ".srt") {
			return filepath.Join(dir, name)
		}
	}
	return ""
}

func (d *Downloader) BurnSubtitles(videoPath, srtPath string) (string, error) {
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", fmt.Errorf("ffmpeg not found in PATH")
	}

	dir := filepath.Dir(videoPath)
	
	// Create safe, generic filenames inside the same output directory
	// to avoid any potential shell/FFmpeg escaping issues with single quotes or spaces in the title.
	safeVideoPath := filepath.Join(dir, "temp_video_input.mp4")
	safeSRTPath := filepath.Join(dir, "temp_video_subs.srt")
	safeOutputPath := filepath.Join(dir, "temp_video_output.mp4")

	// Ensure any old temp files are cleaned up first
	_ = os.Remove(safeVideoPath)
	_ = os.Remove(safeSRTPath)
	_ = os.Remove(safeOutputPath)

	// Rename/move the files to safe paths
	err = os.Rename(videoPath, safeVideoPath)
	if err != nil {
		return "", fmt.Errorf("failed to move video to safe path: %v", err)
	}
	defer func() {
		// Clean up inputs if still there
		_ = os.Remove(safeVideoPath)
		_ = os.Remove(safeSRTPath)
	}()

	err = os.Rename(srtPath, safeSRTPath)
	if err != nil {
		// Move video back if subtitle move failed
		_ = os.Rename(safeVideoPath, videoPath)
		return "", fmt.Errorf("failed to move subtitle to safe path: %v", err)
	}

	absSRT, err := filepath.Abs(safeSRTPath)
	if err != nil {
		_ = os.Rename(safeVideoPath, videoPath)
		return "", fmt.Errorf("failed to get absolute path for subtitles: %v", err)
	}

	// Escape path for Windows drive letter colon and slashes
	escapedSRT := filepath.ToSlash(absSRT)
	escapedSRT = strings.ReplaceAll(escapedSRT, ":", "\\:")

	// Burn subtitles using veryfast preset and copying the audio stream
	cmd := exec.Command("ffmpeg",
		"-i", safeVideoPath,
		"-vf", fmt.Sprintf("subtitles='%s'", escapedSRT),
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a", "copy",
		"-y",
		safeOutputPath,
	)

	fmt.Printf("Burning subtitles into video using safe paths...\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Restore original files if ffmpeg fails
		_ = os.Rename(safeVideoPath, videoPath)
		_ = os.Rename(safeSRTPath, srtPath)
		return "", fmt.Errorf("failed to burn subtitles: %v\nOutput: %s", err, string(output))
	}

	// Rename the burned output back to the original video filename
	err = os.Rename(safeOutputPath, videoPath)
	if err != nil {
		// Restore original video if rename fails
		_ = os.Rename(safeVideoPath, videoPath)
		return "", fmt.Errorf("failed to rename burned video file: %v", err)
	}

	// Clean up raw subtitle file
	_ = os.Remove(srtPath)

	return videoPath, nil
}

