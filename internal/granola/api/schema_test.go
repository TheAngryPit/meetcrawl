package api

import (
	"encoding/json"
	"testing"
)

func TestParseTranscriptItemsSupported(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`[
	  {"speaker":{"source":"microphone","attribution":"me"},"text":"hello","start_time":"2026-01-15T14:00:00Z","end_time":"2026-01-15T14:00:05Z"}
	]`)
	items, err := ParseTranscriptItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Text != "hello" {
		t.Fatalf("items = %+v", items)
	}
}

func TestParseTranscriptItemsUnsupportedSpeakerSource(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`[
	  {"speaker":{"source":"invalid"},"text":"nope","start_time":"2026-01-15T14:00:00Z","end_time":"2026-01-15T14:00:05Z"}
	]`)
	_, err := ParseTranscriptItems(raw)
	if err == nil {
		t.Fatal("expected error")
	}
}
