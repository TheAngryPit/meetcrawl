package api

import (
	"encoding/json"
	"testing"
)

func TestParseTranscriptItemsSupported(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{
	  "transcript": [
	    {"speaker":{"display_name":"Alpha"},"text":"hello","timestamp":"00:00:05"}
	  ]
	}`)
	items, err := ParseTranscriptItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Text != "hello" {
		t.Fatalf("items = %+v", items)
	}
}

func TestParseTranscriptItemsMissingText(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`[
	  {"speaker":{"display_name":"Alpha"},"timestamp":"00:00:05"}
	]`)
	_, err := ParseTranscriptItems(raw)
	if err == nil {
		t.Fatal("expected error")
	}
}
