package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-clipper/internal/downloader"
	"go-clipper/internal/dtos"
	"go-clipper/internal/processor"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// OpenAI rejects audio uploads larger than this.
const openAIMaxAudioBytes = 25 * 1024 * 1024

// OpenAIAnalyzer can't watch videos, so it works from a timestamped transcript:
// YouTube subtitles when the video has them, otherwise an OpenAI transcription.
type OpenAIAnalyzer struct {
	client          *openAIClient
	model           string
	transcribeModel string
}

func (a *OpenAIAnalyzer) AnalyzeVideoUrl(ctx context.Context, url string, payload dtos.AnalyzeRequest) (*dtos.AnalysisResult, error) {
	start := time.Now()
	log.Printf("Analyzing video url with OpenAI (%s): %s", a.model, url)

	lines, err := a.transcript(ctx, url, payload.VideoPath, payload.StartTimestamp, payload.EndTimestamp)
	if err != nil {
		return nil, err
	}

	prompt := analyzePrompt(payload, "Analyze the video using its timestamped transcript below. Each line is [start-end in seconds] followed by what is said.") +
		"\n\nTranscript:\n" + formatTranscript(lines)

	var result dtos.AnalysisResult
	if err := a.client.chatJSON(ctx, a.model, prompt, "clip_analysis", analysisSchema, &result); err != nil {
		return nil, err
	}

	log.Printf("Analyzing video %s latency: %s", url, time.Since(start))
	log.Println("Segments: ", len(result.Segments))
	return &result, nil
}

func (a *OpenAIAnalyzer) GenerateDescription(ctx context.Context, url string) (*dtos.DescriptionResult, error) {
	start := time.Now()

	lines, err := a.transcript(ctx, url, "", "", "")
	if err != nil {
		return nil, err
	}

	intro := fmt.Sprintf("Create caption/description for this youtube video %s. Use its transcript below to understand what the video is about.", url)
	prompt := descriptionPrompt(intro) + "\n\nTranscript:\n" + formatTranscript(lines)

	var result dtos.DescriptionResult
	if err := a.client.chatJSON(ctx, a.model, prompt, "video_description", descriptionSchema, &result); err != nil {
		return nil, err
	}

	log.Printf("Generating description %s latency: %s", url, time.Since(start))
	return &result, nil
}

// transcript returns the video's transcript limited to [startTs, endTs].
// It uses the local video (and its sidecar .srt) when videoPath exists,
// otherwise it fetches subtitles or audio from the URL.
func (a *OpenAIAnalyzer) transcript(ctx context.Context, url, videoPath, startTs, endTs string) ([]processor.SubtitleLine, error) {
	rangeStart, rangeEnd := parseRange(startTs, endTs)

	var srtPath, mediaPath string
	if videoPath != "" {
		if _, err := os.Stat(videoPath); err == nil {
			srtPath = downloader.FindSubtitleFile(videoPath)
			mediaPath = videoPath
		}
	}
	if mediaPath == "" {
		s, audio, cleanup, err := downloader.NewDownloader("", "").DownloadTranscriptSource(url)
		if err != nil {
			return nil, err
		}
		defer cleanup()
		srtPath, mediaPath = s, audio
	}

	var lines []processor.SubtitleLine
	var err error
	if srtPath != "" {
		log.Printf("Using subtitles as transcript: %s", srtPath)
		lines, err = processor.ReadSRTFile(srtPath)
	} else {
		log.Printf("Transcribing audio with OpenAI (%s)", a.transcribeModel)
		lines, err = a.client.transcribe(ctx, "transcriptions", a.transcribeModel, mediaPath, rangeStart, rangeEnd)
	}
	if err != nil {
		return nil, err
	}

	var inRange []processor.SubtitleLine
	for _, l := range lines {
		if l.End > rangeStart && l.Start < rangeEnd {
			inRange = append(inRange, l)
		}
	}
	if len(inRange) == 0 {
		return nil, fmt.Errorf("transcript is empty for the requested range")
	}
	return inRange, nil
}

// OpenAICaptioner writes English subtitles using OpenAI's translation endpoint,
// which transcribes any spoken language straight into English SRT.
type OpenAICaptioner struct {
	client          *openAIClient
	transcribeModel string
}

func (c *OpenAICaptioner) GenerateSRT(ctx context.Context, videoPath string) (string, error) {
	log.Printf("Generating SRT with OpenAI (%s) for: %s", c.transcribeModel, videoPath)

	audioPath, cleanup, err := prepareAudio(videoPath, 0, math.Inf(1))
	if err != nil {
		return "", err
	}
	defer cleanup()

	srt, err := c.client.audioSRT(ctx, "translations", c.transcribeModel, audioPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(srt), nil
}

// formatTranscript renders lines as "[12.3-15.0] text", dropping the repeated
// rows that YouTube's auto-generated captions carry over between cues.
func formatTranscript(lines []processor.SubtitleLine) string {
	var sb strings.Builder
	lastRow := ""
	for _, l := range lines {
		var rows []string
		for _, row := range strings.Split(l.Text, "\n") {
			row = strings.TrimSpace(row)
			if row == "" || row == lastRow {
				continue
			}
			rows = append(rows, row)
			lastRow = row
		}
		if len(rows) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "[%.1f-%.1f] %s\n", l.Start, l.End, strings.Join(rows, " "))
	}
	return sb.String()
}

// parseRange converts the request's start/end timestamps to seconds.
// Empty or non-numeric values (e.g. "start"/"end") mean the whole video.
func parseRange(startTs, endTs string) (float64, float64) {
	start, err := processor.ParseTimestamp(startTs)
	if err != nil || start < 0 {
		start = 0
	}
	end, err := processor.ParseTimestamp(endTs)
	if err != nil || end <= start {
		end = math.Inf(1)
	}
	return start, end
}

// prepareAudio converts a media file to small mono MP3 (optionally trimmed to
// [start, end]) so it fits OpenAI's upload limit.
func prepareAudio(mediaPath string, start, end float64) (string, func(), error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", nil, fmt.Errorf("ffmpeg not found in PATH")
	}

	tmp, err := os.CreateTemp("", "go-clipper-*.mp3")
	if err != nil {
		return "", nil, err
	}
	_ = tmp.Close()
	cleanup := func() { _ = os.Remove(tmp.Name()) }

	args := []string{"-y"}
	if start > 0 {
		args = append(args, "-ss", strconv.FormatFloat(start, 'f', 3, 64))
	}
	args = append(args, "-i", mediaPath)
	if !math.IsInf(end, 1) {
		args = append(args, "-t", strconv.FormatFloat(end-start, 'f', 3, 64))
	}
	args = append(args, "-vn", "-ac", "1", "-ar", "16000", "-b:a", "32k", tmp.Name())

	if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to extract audio: %v\nOutput: %s", err, string(output))
	}

	info, err := os.Stat(tmp.Name())
	if err != nil {
		cleanup()
		return "", nil, err
	}
	if info.Size() > openAIMaxAudioBytes {
		cleanup()
		return "", nil, fmt.Errorf("audio is %.1f MB, over OpenAI's 25 MB limit (about 1h40m of speech); set start_timestamp/end_timestamp to analyze a shorter range",
			float64(info.Size())/1024/1024)
	}
	return tmp.Name(), cleanup, nil
}

var analysisSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"description": map[string]any{"type": "string"},
		"segments": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"start":       map[string]any{"type": "string"},
					"end":         map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"hook":        map[string]any{"type": "string"},
				},
				"required":             []string{"start", "end", "description", "hook"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"description", "segments"},
	"additionalProperties": false,
}

var descriptionSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"description": map[string]any{"type": "string"},
		"tags":        map[string]any{"type": "string"},
	},
	"required":             []string{"description", "tags"},
	"additionalProperties": false,
}

// openAIClient is a minimal client for the OpenAI REST API (or any compatible API via OPENAI_BASE_URL).
type openAIClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func newOpenAIClient(apiKey, baseURL string) *openAIClient {
	return &openAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Minute},
	}
}

// chatJSON sends a single user prompt and decodes the structured JSON reply into out.
func (c *openAIClient) chatJSON(ctx context.Context, model, prompt, schemaName string, schema map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   schemaName,
				"strict": true,
				"schema": schema,
			},
		},
	})
	if err != nil {
		return err
	}

	respBody, err := c.do(ctx, "/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	var resp struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("failed to decode OpenAI response: %v", err)
	}
	if len(resp.Choices) == 0 {
		return fmt.Errorf("no choices returned from OpenAI")
	}

	choice := resp.Choices[0]
	log.Printf("Finish Reason: %s", choice.FinishReason)
	if choice.Message.Refusal != "" {
		return fmt.Errorf("OpenAI refused: %s", choice.Message.Refusal)
	}
	log.Printf("Raw Analysis Output: %s", choice.Message.Content)

	if err := json.Unmarshal([]byte(choice.Message.Content), out); err != nil {
		return fmt.Errorf("failed to parse JSON: %v\nRaw: %s", err, choice.Message.Content)
	}
	return nil
}

// transcribe turns a media file into subtitle lines with timestamps relative
// to the original media (offset by start when only a range is transcribed).
func (c *openAIClient) transcribe(ctx context.Context, endpoint, model, mediaPath string, start, end float64) ([]processor.SubtitleLine, error) {
	audioPath, cleanup, err := prepareAudio(mediaPath, start, end)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	srt, err := c.audioSRT(ctx, endpoint, model, audioPath)
	if err != nil {
		return nil, err
	}

	lines := processor.ParseSRT(srt)
	for i := range lines {
		lines[i].Start += start
		lines[i].End += start
	}
	return lines, nil
}

// audioSRT uploads audio to /audio/{endpoint} ("transcriptions" or "translations") and returns SRT text.
func (c *openAIClient) audioSRT(ctx context.Context, endpoint, model, audioPath string) (string, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", model)
	_ = mw.WriteField("response_format", "srt")
	part, err := mw.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	respBody, err := c.do(ctx, "/audio/"+endpoint, mw.FormDataContentType(), &buf)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
}

func (c *openAIClient) do(ctx context.Context, path, contentType string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OpenAI %s returned %s: %s", path, resp.Status, string(respBody))
	}
	return respBody, nil
}
