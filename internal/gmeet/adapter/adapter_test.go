package adapter_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/adapter"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func TestAdapterImplementsContract(t *testing.T) {
	t.Parallel()
	repo := mustRepoRoot(t)
	fixtureDir := filepath.Join(repo, "testdata", "fixtures", "gdrive", "supported")
	cfg := config.Config{
		Version:  1,
		DBPath:   filepath.Join(t.TempDir(), "gmeetcrawl.db"),
		CacheDir: filepath.Join(t.TempDir(), "cache"),
		LogDir:   filepath.Join(t.TempDir(), "logs"),
	}
	var _ source.Adapter = adapter.Adapter{}
	got, err := adapter.Adapter{
		Config:         cfg,
		FixtureDir:     fixtureDir,
		CrawlerVersion: "gmeetcrawl-test",
	}.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync() = %v", err)
	}
	if got.Code != source.OutcomeOK {
		t.Fatalf("Code = %q", got.Code)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
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
