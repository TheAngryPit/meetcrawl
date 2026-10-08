package split_test

import (
	"strings"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/split"
)

func TestParseMarkdownEmojiHeadingsWin(t *testing.T) {
	t.Parallel()
	md := `# **📝 Observações**

### Resumo
Emoji markdown notes body.

# **📖 Transcrição**

### 14:05:00
Speaker Alpha: emoji markdown transcript.
`
	plain := "broken plain without tab markers\n"
	res, format, err := split.ParseWithFormat(md, plain)
	if err != nil {
		t.Fatalf("ParseWithFormat() = %v", err)
	}
	if format != "markdown" {
		t.Fatalf("format = %q, want markdown", format)
	}
	if !strings.Contains(res.TranscriptText, "emoji markdown transcript") {
		t.Fatalf("transcript = %q", res.TranscriptText)
	}
	if _, err := split.Parse("", plain); err == nil {
		t.Fatal("plain-only path should fail on broken plain fixture")
	}
}

func TestParseMarkdownBothTabs(t *testing.T) {
	t.Parallel()
	md := `# **Observações**

## Reunião em 15 de jan. de 2026 às 14:00 WET

### Resumo
Blorp flindle rekord za reunião sintética.

# **Transcrição**

## Reunião em 15 de jan. de 2026 às 14:00 WET

### 14:05:00
Speaker Alpha: flarn dialogue.
`
	res, err := split.Parse(md, "")
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}
	if !strings.Contains(res.NotesText, "reunião sintética") {
		t.Fatalf("notes = %q", res.NotesText)
	}
	if !strings.Contains(res.TranscriptText, "Speaker Alpha") {
		t.Fatalf("transcript = %q", res.TranscriptText)
	}
	if len(res.Flags) != 0 {
		t.Fatalf("flags = %v", res.Flags)
	}
}

func TestParsePlainBothTabsCRLF(t *testing.T) {
	t.Parallel()
	plain := "\ufeff📝 Observações\r\n\r\n2026/01/15\r\nReunião em 15 de jan. de 2026\r\nResumo\r\nBlorp\r\n\r\n📖 Transcrição\r\n\r\nReunião em 15 de jan. de 2026\r\n14:05:00\r\nSpeaker Alpha: flarn\r\n"
	res, err := split.Parse("", plain)
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}
	if res.TranscriptText == "" {
		t.Fatal("expected transcript section")
	}
}

func TestParseNotesOnly(t *testing.T) {
	t.Parallel()
	md := `# **Notes by Gemini**

### Summary
Only notes tab here.
`
	res, err := split.Parse(md, "")
	if err != nil {
		t.Fatalf("Parse() = %v", err)
	}
	if res.TranscriptText != "" {
		t.Fatalf("transcript = %q", res.TranscriptText)
	}
	if len(res.Flags) != 1 || res.Flags[0] != split.FlagNotesOnly {
		t.Fatalf("flags = %v", res.Flags)
	}
}

func TestParseFailClosed(t *testing.T) {
	t.Parallel()
	if _, err := split.Parse("", "no markers here"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := split.Parse("", "# **Transcrição**\nonly transcript"); err == nil {
		t.Fatal("expected error for missing notes marker")
	}
}
