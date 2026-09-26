package downloader

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// videoFilter is a gallery-dl filter that skips the video slides of a post.
const videoFilter = "extension not in ('mp4', 'm4v', 'mov', 'webm')"

// IsInstagramURL reports whether rawURL points at instagram.com.
func IsInstagramURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "instagram.com" || strings.HasSuffix(host, ".instagram.com")
}

// DownloadInstagramImages downloads the images of an Instagram post, including
// every slide of a carousel, and returns the saved file paths in slide order.
// yt-dlp only handles video posts, so this uses gallery-dl. Set includeVideos
// to also keep the video slides of a mixed carousel.
func (d *Downloader) DownloadInstagramImages(postURL string, includeVideos bool) ([]string, error) {
	if !IsInstagramURL(postURL) {
		return nil, fmt.Errorf("not an Instagram URL: %s", postURL)
	}

	if _, err := exec.LookPath("gallery-dl"); err != nil {
		return nil, fmt.Errorf("gallery-dl not found in PATH. Please install it with 'pip install gallery-dl'")
	}

	outputDir := filepath.Join(d.OutputDir, "instagram")
	args := []string{
		"-D", outputDir,
		"-f", "{username}_{shortcode}_{num}.{extension}",
	}
	if !includeVideos {
		args = append(args, "--filter", videoFilter)
	}
	// Instagram hides most posts from logged-out requests
	if d.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", d.CookiesBrowser)
	}
	args = append(args, postURL)

	cmd := exec.Command("gallery-dl", args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	fmt.Printf("Downloading Instagram images: %s\n", postURL)

	err := cmd.Run()
	paths := parseGalleryDLOutput(stdoutBuf.String())

	if len(paths) == 0 {
		if err != nil {
			return nil, fmt.Errorf("failed to download Instagram images: %v\nStderr: %s", err, stderrBuf.String())
		}
		return nil, fmt.Errorf("no images found in post. Stderr: %s", stderrBuf.String())
	}
	if err != nil {
		// gallery-dl exits non-zero when any single file fails; keep what we got
		fmt.Printf("Warning: some Instagram files failed to download: %v\n", err)
	}
	return paths, nil
}

// parseGalleryDLOutput returns the file paths gallery-dl printed to stdout.
// Files that already existed are printed with a "# " prefix and are included too.
func parseGalleryDLOutput(stdout string) []string {
	paths := make([]string, 0)
	for _, line := range strings.Split(stdout, "\n") {
		path := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "# "))
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}
