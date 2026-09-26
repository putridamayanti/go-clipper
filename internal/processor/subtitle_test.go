package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCutSubtitle(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Video.en.srt")
	srt := string(rune(0xFEFF)) + "1\r\n00:00:05,000 --> 00:00:09,500\r\nBefore and into the clip\r\n\r\n" +
		"2\r\n00:00:12,250 --> 00:00:14,000\r\nInside\r\nsecond line\r\n\r\n" +
		"3\r\n00:00:19,000 --> 00:00:25,000\r\nCrosses the end\r\n\r\n" +
		"4\r\n00:00:30,000 --> 00:00:31,000\r\nAfter\r\n"
	if err := os.WriteFile(src, []byte(srt), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewProcessor(dir, "", false)
	out, err := p.CutSubtitle(src, filepath.Join(dir, "hook.mp4"), "8", "20")
	if err != nil {
		t.Fatal(err)
	}
	if out != filepath.Join(dir, "hook.srt") {
		t.Fatalf("unexpected path %s", out)
	}

	got, _ := os.ReadFile(out)
	want := "1\n00:00:00,000 --> 00:00:01,500\nBefore and into the clip\n\n" +
		"2\n00:00:04,250 --> 00:00:06,000\nInside\nsecond line\n\n" +
		"3\n00:00:11,000 --> 00:00:12,000\nCrosses the end\n\n"
	if string(got) != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestCutSubtitleNoOverlap(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Video.srt")
	os.WriteFile(src, []byte("1\n00:00:01,000 --> 00:00:02,000\nHi\n"), 0644)

	out, err := NewProcessor(dir, "", false).CutSubtitle(src, filepath.Join(dir, "c.mp4"), "60", "90")
	if err != nil || out != "" {
		t.Fatalf("expected no file, got %q, %v", out, err)
	}
}

func TestParseTimestamp(t *testing.T) {
	cases := map[string]float64{"75": 75, "75.5": 75.5, "01:15": 75, "00:01:15.250": 75.25}
	for in, want := range cases {
		if got, err := ParseTimestamp(in); err != nil || got != want {
			t.Errorf("ParseTimestamp(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
}
