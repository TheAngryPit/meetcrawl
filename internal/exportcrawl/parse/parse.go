package parse

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/TheAngryPit/meetcrawl/internal/source"
)

type Format int

const (
	FormatVTT Format = iota + 1
	FormatSRT
	FormatTXT
	FormatMD
)

func FormatFromPath(path string) (Format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".vtt":
		return FormatVTT, nil
	case ".srt":
		return FormatSRT, nil
	case ".txt":
		return FormatTXT, nil
	case ".md", ".markdown":
		return FormatMD, nil
	default:
		return 0, fmt.Errorf("unsupported export file extension %q", filepath.Ext(path))
	}
}

func Normalize(format Format, raw []byte) (string, source.Fidelity, error) {
	if !utf8.Valid(raw) {
		return "", 0, fmt.Errorf("export file is not valid utf-8")
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	switch format {
	case FormatVTT:
		out, err := normalizeVTT(text)
		if err != nil {
			return "", 0, err
		}
		return out, source.FidelityTranscript, nil
	case FormatSRT:
		out, err := normalizeSRT(text)
		if err != nil {
			return "", 0, err
		}
		return out, source.FidelityTranscript, nil
	case FormatTXT:
		out := strings.TrimSpace(text)
		if out == "" {
			return "", 0, fmt.Errorf("empty text export")
		}
		return out, source.FidelityNotes, nil
	case FormatMD:
		out := strings.TrimSpace(text)
		if out == "" {
			return "", 0, fmt.Errorf("empty markdown export")
		}
		return out, source.FidelityNotes, nil
	default:
		return "", 0, fmt.Errorf("unknown export format")
	}
}

func normalizeVTT(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty vtt export")
	}
	if !strings.HasPrefix(trimmed, "WEBVTT") {
		return "", fmt.Errorf("vtt missing WEBVTT header")
	}
	lines := strings.Split(trimmed, "\n")
	var cues []string
	var block []string
	flush := func() {
		if len(block) == 0 {
			return
		}
		cueText := strings.TrimSpace(strings.Join(block, "\n"))
		if cueText != "" {
			cues = append(cues, cueText)
		}
		block = block[:0]
	}
	for i, line := range lines {
		if i == 0 {
			continue
		}
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if strings.Contains(line, "-->") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "NOTE") {
			continue
		}
		block = append(block, strings.TrimSpace(line))
	}
	flush()
	if len(cues) == 0 {
		return "", fmt.Errorf("vtt has no cue text")
	}
	return strings.Join(cues, "\n"), nil
}

func normalizeSRT(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty srt export")
	}
	blocks := strings.Split(trimmed, "\n\n")
	var cues []string
	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 2 {
			continue
		}
		start := 0
		if idx := strings.Index(lines[0], "-->"); idx == -1 && len(lines) > 1 {
			if strings.Contains(lines[1], "-->") {
				start = 1
			}
		}
		if start >= len(lines)-1 {
			continue
		}
		if !strings.Contains(lines[start], "-->") {
			return "", fmt.Errorf("malformed srt timestamp block")
		}
		text := strings.TrimSpace(strings.Join(lines[start+1:], "\n"))
		if text != "" {
			cues = append(cues, text)
		}
	}
	if len(cues) == 0 {
		return "", fmt.Errorf("srt has no cue text")
	}
	return strings.Join(cues, "\n"), nil
}
