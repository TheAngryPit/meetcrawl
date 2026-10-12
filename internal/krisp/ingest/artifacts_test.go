package ingest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/krisp/api"
	"github.com/TheAngryPit/meetcrawl/internal/krisp/ingest"
)

type emptyListClient struct{}

func (emptyListClient) ListMeetingsForSync(context.Context) ([]api.MeetingForSync, error) {
	return nil, nil
}

func (emptyListClient) TranscriptItems(context.Context, string) ([]api.TranscriptItem, error) {
	return nil, nil
}

func TestBuildArtifactsEmptyListFailsClosed(t *testing.T) {
	t.Parallel()
	_, err := ingest.BuildArtifacts(context.Background(), emptyListClient{}, ingest.Options{CrawlerVersion: "test"})
	if err == nil {
		t.Fatal("BuildArtifacts() err = nil, want failure")
	}
	if !strings.Contains(err.Error(), "no krisp meetings returned") {
		t.Fatalf("BuildArtifacts() err = %v", err)
	}
}
