package provider_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestListEventsAuthPresentWarnsOnClientFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	clientPath := filepath.Join(dir, "oauth-client.json")
	tokenPath := filepath.Join(dir, "token.json")
	if err := os.WriteFile(clientPath, []byte("not-valid-oauth-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tokenPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	var warn bytes.Buffer
	events, err := provider.ListEvents(context.Background(), provider.Options{
		OAuthClientPath: clientPath,
		TokenPath:       tokenPath,
		Warn:            &warn,
	})
	if err != nil {
		t.Fatalf("ListEvents() err = %v", err)
	}
	if events != nil {
		t.Fatalf("ListEvents() = %#v, want nil on client failure", events)
	}
	msg := warn.String()
	if msg == "" {
		t.Fatal("expected warning on stderr when auth files exist but client fails")
	}
	if strings.Contains(msg, "token") && strings.Contains(strings.ToLower(msg), "secret") {
		t.Fatalf("warning must not include secrets: %q", msg)
	}
	if !strings.Contains(msg, "calendar enrichment skipped") {
		t.Fatalf("warning = %q, want skip prefix", msg)
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
