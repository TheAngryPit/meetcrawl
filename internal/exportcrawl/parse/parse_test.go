package parse_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/parse"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func TestNormalizeVTTAndSRT(t *testing.T) {
	t.Parallel()
	vtt := []byte("WEBVTT\n\n00:00:01.000 --> 00:00:04.000\nSynthetic standup transcript for fixture ingest.\n")
	got, fidelity, err := parse.Normalize(parse.FormatVTT, vtt)
	if err != nil {
		t.Fatalf("VTT: %v", err)
	}
	if fidelity != source.FidelityTranscript {
		t.Fatalf("fidelity = %v", fidelity)
	}
	want := "Synthetic standup transcript for fixture ingest."
	if got != want {
		t.Fatalf("VTT text = %q, want %q", got, want)
	}

	srt := []byte("1\n00:00:01,000 --> 00:00:04,000\nSynthetic standup transcript for fixture ingest.\n")
	got, fidelity, err = parse.Normalize(parse.FormatSRT, srt)
	if err != nil {
		t.Fatalf("SRT: %v", err)
	}
	if got != want {
		t.Fatalf("SRT text = %q, want %q", got, want)
	}
}

func TestMalformedVTTFailsClosed(t *testing.T) {
	t.Parallel()
	_, _, err := parse.Normalize(parse.FormatVTT, []byte("NOT WEBVTT\n"))
	if err == nil {
		t.Fatal("expected error for malformed vtt")
	}
}
