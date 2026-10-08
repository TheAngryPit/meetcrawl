package source_test

import (
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func TestProvenanceHashStable(t *testing.T) {
	t.Parallel()
	in := source.ProvenanceInput{
		Source:         "openwhispr",
		SourceID:       "note-synthetic-001",
		SourceRevision: "rev-1",
		ContentHash:    source.ContentHash("same text"),
		CrawlerVersion: "whispcrawl-test-0.0.0",
	}
	a, err := source.ProvenanceHash(in)
	if err != nil {
		t.Fatalf("ProvenanceHash() = %v", err)
	}
	b, err := source.ProvenanceHash(in)
	if err != nil {
		t.Fatalf("ProvenanceHash() = %v", err)
	}
	if a != b {
		t.Fatalf("hash changed: %q vs %q", a, b)
	}
	if in.SourceRevision != "rev-1" {
		t.Fatalf("revision mutated")
	}
}

func TestContentHashDiffersForText(t *testing.T) {
	t.Parallel()
	a := source.ContentHash("alpha")
	b := source.ContentHash("beta")
	if a == b {
		t.Fatal("expected different content hashes")
	}
}
