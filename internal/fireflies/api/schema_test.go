package api

import (
	"encoding/json"
	"testing"
)

func TestParseSentencesSupported(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`[
	  {"index":0,"speaker_name":"Alpha","text":"hello","start_time":1.5,"end_time":4.0}
	]`)
	items, err := ParseSentences(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Text != "hello" {
		t.Fatalf("items = %+v", items)
	}
}

func TestParseSentencesMissingText(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`[
	  {"speaker_name":"Alpha","start_time":1.0,"end_time":2.0}
	]`)
	_, err := ParseSentences(raw)
	if err == nil {
		t.Fatal("expected error")
	}
}
