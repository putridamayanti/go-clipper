package analyzer

import (
	"context"
	"encoding/json"
	"go-clipper/internal/dtos"
	"go-clipper/internal/processor"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChatJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("unexpected request %s auth=%q", r.URL.Path, r.Header.Get("Authorization"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "my-model" {
			t.Errorf("model = %v", body["model"])
		}
		content := `{"description":"d","segments":[{"start":"10","end":"40","description":"x","hook":"Hook"}]}`
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": content}}},
		})
	}))
	defer srv.Close()

	var out dtos.AnalysisResult
	err := newOpenAIClient("key", srv.URL).chatJSON(context.Background(), "my-model", "p", "clip_analysis", analysisSchema, &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Segments) != 1 || out.Segments[0].Hook != "Hook" || out.Segments[0].Start != "10" {
		t.Fatalf("unexpected result %+v", out)
	}
}

func TestChatJSONError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad model"}}`, http.StatusNotFound)
	}))
	defer srv.Close()

	var out dtos.AnalysisResult
	err := newOpenAIClient("key", srv.URL).chatJSON(context.Background(), "m", "p", "s", analysisSchema, &out)
	if err == nil || !strings.Contains(err.Error(), "bad model") {
		t.Fatalf("expected API error, got %v", err)
	}
}

func TestAudioSRT(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/translations" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.FormValue("model") != "whisper-1" || r.FormValue("response_format") != "srt" {
			t.Errorf("form = %v", r.Form)
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(f)
		if string(data) != "audio" {
			t.Errorf("file = %q", data)
		}
		_, _ = w.Write([]byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"))
	}))
	defer srv.Close()

	audio := filepath.Join(t.TempDir(), "a.mp3")
	_ = os.WriteFile(audio, []byte("audio"), 0644)

	srt, err := newOpenAIClient("key", srv.URL).audioSRT(context.Background(), "translations", "whisper-1", audio)
	if err != nil || !strings.Contains(srt, "Hello") {
		t.Fatalf("got %q, %v", srt, err)
	}
}

func TestFormatTranscriptDropsRollingDuplicates(t *testing.T) {
	lines := []processor.SubtitleLine{
		{Start: 0, End: 2, Text: "hello there"},
		{Start: 2, End: 4, Text: "hello there\nhow are you"},
		{Start: 4, End: 6, Text: "how are you\nfine"},
	}
	want := "[0.0-2.0] hello there\n[2.0-4.0] how are you\n[4.0-6.0] fine\n"
	if got := formatTranscript(lines); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParseRange(t *testing.T) {
	if s, e := parseRange("start", "end"); s != 0 || !math.IsInf(e, 1) {
		t.Errorf("defaults: %v %v", s, e)
	}
	if s, e := parseRange("60", "02:00"); s != 60 || e != 120 {
		t.Errorf("range: %v %v", s, e)
	}
}
