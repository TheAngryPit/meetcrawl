package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/calendar/provider"
)

func TestListEventsNoAuthDegrades(t *testing.T) {
	t.Parallel()
	events, err := provider.ListEvents(context.Background(), provider.Options{
		OAuthClientPath: filepath.Join(t.TempDir(), "missing-client.json"),
		TokenPath:       filepath.Join(t.TempDir(), "missing-token.json"),
	})
	if err != nil {
		t.Fatalf("ListEvents() err = %v", err)
	}
	if events != nil {
		t.Fatalf("ListEvents() = %#v, want nil without auth", events)
	}
}

func TestListEventsFixtureDir(t *testing.T) {
	t.Parallel()
	repo := mustRepoRoot(t)
	dir := filepath.Join(repo, "testdata", "fixtures", "calendar", "synthetic")
	events, err := provider.ListEvents(context.Background(), provider.Options{FixtureDir: dir})
	if err != nil {
		t.Fatalf("ListEvents() err = %v", err)
	}
	if len(events) != 1 || events[0].ICalUID != "cal-synthetic-001" {
		t.Fatalf("events = %#v, want one synthetic event", events)
	}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
