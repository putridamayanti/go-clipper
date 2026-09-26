package processor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CutSubtitle slices the source .srt to the [start, end] range of a clip and
// writes it next to the clip as <clip>.srt, with timestamps shifted so the
// clip starts at 00:00:00. Returns "" (no error) when no line falls in range.
func (p *Processor) CutSubtitle(sourceSRTPath, clipPath, start, end string) (string, error) {
	startSec, err := ParseTimestamp(start)
	if err != nil {
		return "", fmt.Errorf("invalid start %q: %v", start, err)
	}
	endSec, err := ParseTimestamp(end)
	if err != nil {
		return "", fmt.Errorf("invalid end %q: %v", end, err)
	}

	lines, err := ReadSRTFile(sourceSRTPath)
	if err != nil {
		return "", err
	}

	var clipLines []SubtitleLine
	for _, l := range lines {
		// Keep any line that overlaps the clip, trimmed to the clip bounds
		if l.End <= startSec || l.Start >= endSec {
			continue
		}
		s := max(l.Start, startSec) - startSec
		e := min(l.End, endSec) - startSec
		if e <= s {
			continue
		}
		clipLines = append(clipLines, SubtitleLine{Start: s, End: e, Text: l.Text})
	}

	if len(clipLines) == 0 {
		return "", nil
	}

	srtPath := strings.TrimSuffix(clipPath, ".mp4") + ".srt"
	if err := p.generateSRT(srtPath, clipLines); err != nil {
		return "", fmt.Errorf("failed to write clip SRT: %v", err)
	}
	return srtPath, nil
}

func ReadSRTFile(path string) ([]SubtitleLine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read SRT: %v", err)
	}
	return ParseSRT(string(data)), nil
}

// ParseSRT parses SRT content into lines, skipping malformed blocks.
func ParseSRT(content string) []SubtitleLine {
	content = strings.TrimPrefix(content, string(rune(0xFEFF)))
	content = strings.ReplaceAll(content, "\r\n", "\n")

	var lines []SubtitleLine
	for _, block := range strings.Split(content, "\n\n") {
		rows := strings.Split(strings.TrimSpace(block), "\n")
		// Find the "00:00:01,000 --> 00:00:02,000" row; the index row before it is optional
		timeIdx := -1
		for i, row := range rows {
			if strings.Contains(row, "-->") {
				timeIdx = i
				break
			}
		}
		if timeIdx == -1 {
			continue
		}

		parts := strings.SplitN(rows[timeIdx], "-->", 2)
		start, err1 := parseSRTTime(parts[0])
		// Drop any position settings after the end time
		endFields := strings.Fields(parts[1])
		if err1 != nil || len(endFields) == 0 {
			continue
		}
		end, err2 := parseSRTTime(endFields[0])
		if err2 != nil {
			continue
		}

		text := strings.TrimSpace(strings.Join(rows[timeIdx+1:], "\n"))
		if text == "" {
			continue
		}
		lines = append(lines, SubtitleLine{Start: start, End: end, Text: text})
	}
	return lines
}

// parseSRTTime parses "HH:MM:SS,mmm" (also accepts "." as the ms separator).
func parseSRTTime(s string) (float64, error) {
	return ParseTimestamp(strings.ReplaceAll(strings.TrimSpace(s), ",", "."))
}

// ParseTimestamp accepts plain seconds ("75", "75.5") or "MM:SS" / "HH:MM:SS(.ms)".
func ParseTimestamp(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var total float64
	for _, part := range strings.Split(s, ":") {
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0, err
		}
		total = total*60 + v
	}
	return total, nil
}
