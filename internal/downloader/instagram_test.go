package downloader

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIsInstagramURL(t *testing.T) {
	cases := map[string]bool{
		"https://www.instagram.com/p/C1a2b3c4d5e/":         true,
		"https://instagram.com/p/C1a2b3c4d5e/?img_index=2": true,
		"https://m.instagram.com/reel/C1a2b3c4d5e/":        true,
		"https://www.youtube.com/watch?v=abc":              false,
		"https://notinstagram.com/p/abc/":                  false,
		"not a url":                                        false,
	}
	for in, want := range cases {
		if got := IsInstagramURL(in); got != want {
			t.Errorf("IsInstagramURL(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseGalleryDLOutput(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "user_abc_1.jpg")
	second := filepath.Join(dir, "user_abc_2.jpg")
	for _, p := range []string{first, second} {
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	stdout := first + "\r\n" +
		"# " + second + "\n" + // already downloaded
		filepath.Join(dir, "missing.jpg") + "\n" +
		"\n"

	got := parseGalleryDLOutput(stdout)
	want := []string{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseGalleryDLOutput() = %v, want %v", got, want)
	}
}
