package ingest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/fireflies/api"
	"github.com/TheAngryPit/meetcrawl/internal/fireflies/ingest"
)

type emptyListClient struct{}

func (emptyListClient) ListTranscriptsForSync(context.Context) ([]api.TranscriptForSync, error) {
	return nil, nil
}

func (emptyListClient) Sentences(context.Context, string) ([]api.Sentence, error) {
	return nil, nil
}

func TestBuildArtifactsEmptyListFailsClosed(t *testing.T) {
	t.Parallel()
	_, err := ingest.BuildArtifacts(context.Background(), emptyListClient{}, ingest.Options{CrawlerVersion: "test"})
	if err == nil {
		t.Fatal("BuildArtifacts() err = nil, want failure")
	}
	if !strings.Contains(err.Error(), "no fireflies transcripts returned") {
		t.Fatalf("BuildArtifacts() err = %v", err)
	}
}
