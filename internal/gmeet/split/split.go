package split

import (
	"fmt"
	"regexp"
	"strings"
)

const FlagNotesOnly = "notes-only"

type Result struct {
	NotesText      string
	TranscriptText string
	Flags          []string
}

var (
	markdownNotesHeading = regexp.MustCompile(`(?im)^#\s*\*{0,2}\s*(Observações|Notes(?:\s+by\s+Gemini)?)\s*\*{0,2}\s*$`)
	markdownTranscript   = regexp.MustCompile(`(?im)^#\s*\*{0,2}\s*(Transcrição|Transcript)\s*\*{0,2}\s*$`)
	plainNotesLine       = regexp.MustCompile(`(?im)^\s*(?:[\p{So}\p{Sk}]\s*)?(Observações|Notes(?:\s+by\s+Gemini)?)\s*$`)
	plainTranscriptLine  = regexp.MustCompile(`(?im)^(?:[\p{So}\p{Sk}]\s*)?(Transcrição|Transcript)\s*$`)
)

// Parse prefers markdown export text and falls back to text/plain.
func Parse(markdown, plain string) (Result, error) {
	if body := strings.TrimSpace(markdown); body != "" {
		if res, err := parseMarkdown(body); err == nil {
			return res, nil
		}
	}
	if body := strings.TrimSpace(plain); body != "" {
		return parsePlain(body)
	}
	return Result{}, fmt.Errorf("gemini export: empty body")
}

func parseMarkdown(body string) (Result, error) {
	notesIdx := findIndex(markdownNotesHeading, body)
	transIdx := findIndex(markdownTranscript, body)
	return finalizeSplit(body, notesIdx, transIdx, markdownNotesHeading, markdownTranscript)
}

func parsePlain(body string) (Result, error) {
	body = strings.TrimPrefix(body, "\ufeff")
	body = strings.ReplaceAll(body, "\r\n", "\n")
	notesIdx := findIndex(plainNotesLine, body)
	transIdx := findIndex(plainTranscriptLine, body)
	return finalizeSplit(body, notesIdx, transIdx, plainNotesLine, plainTranscriptLine)
}

func findIndex(re *regexp.Regexp, body string) int {
	loc := re.FindStringIndex(body)
	if loc == nil {
		return -1
	}
	return loc[0]
}

func finalizeSplit(body string, notesIdx, transIdx int, notesRe, transRe *regexp.Regexp) (Result, error) {
	if notesIdx < 0 && transIdx < 0 {
		return Result{}, fmt.Errorf("unsupported_schema: missing gemini tab markers")
	}
	if notesIdx < 0 {
		return Result{}, fmt.Errorf("unsupported_schema: missing notes tab marker")
	}
	var res Result
	if transIdx >= 0 && transIdx > notesIdx {
		res.NotesText = strings.TrimSpace(body[notesIdx:transIdx])
		res.TranscriptText = strings.TrimSpace(body[transIdx:])
	} else {
		res.NotesText = strings.TrimSpace(body[notesIdx:])
		res.Flags = append(res.Flags, FlagNotesOnly)
	}
	res.NotesText = stripMarkerLine(res.NotesText, notesRe)
	if res.TranscriptText != "" {
		res.TranscriptText = stripMarkerLine(res.TranscriptText, transRe)
	}
	if res.NotesText == "" {
		return Result{}, fmt.Errorf("unsupported_schema: empty notes section")
	}
	return res, nil
}

func stripMarkerLine(section string, re *regexp.Regexp) string {
	section = strings.TrimSpace(section)
	loc := re.FindStringIndex(section)
	if loc == nil || loc[0] != 0 {
		return section
	}
	return strings.TrimSpace(section[loc[1]:])
}
