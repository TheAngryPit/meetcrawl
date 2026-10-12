package sync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/pocket/archive"
	"github.com/TheAngryPit/meetcrawl/internal/pocket/config"
	"github.com/TheAngryPit/meetcrawl/internal/pocket/sync"
	"github.com/TheAngryPit/meetcrawl/internal/secret"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func TestSyncSupportedFixtureStoresArtifact(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "pocket", "supported")
	cfg := config.Config{
		Version:  1,
		DBPath:   filepath.Join(root, "pocketcrawl.db"),
		CacheDir: filepath.Join(root, "cache"),
		LogDir:   filepath.Join(root, "logs"),
	}
	result, err := sync.Run(context.Background(), cfg, sync.Options{
		FixtureDir:     fixtureDir,
		CrawlerVersion: "pocketcrawl-test",
		Secrets:        secret.MapProvider{},
	})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if result.Code != source.OutcomeOK || result.Artifacts != 1 {
		t.Fatalf("result = %+v", result)
	}
	store, err := archive.Open(context.Background(), cfg.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var sourceID string
	err = store.DB().QueryRowContext(context.Background(), `
select source_id from artifacts where source_id = 'rec_pocket_fixture_standup_001'
`).Scan(&sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if sourceID != "rec_pocket_fixture_standup_001" {
		t.Fatalf("source_id = %q", sourceID)
	}
}

func TestMissingSecretFailsClosed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cfg := config.Config{
		Version:   1,
		DBPath:    filepath.Join(root, "pocketcrawl.db"),
		CacheDir:  filepath.Join(root, "cache"),
		LogDir:    filepath.Join(root, "logs"),
		APIKeyRef: secret.AccountPocketAPIKey,
	}
	result, err := sync.Run(context.Background(), cfg, sync.Options{
		CrawlerVersion: "pocketcrawl-test",
		Secrets:        secret.MapProvider{},
	})
	if err == nil {
		t.Fatal("Run() err = nil, want failure")
	}
	if result.Code != source.OutcomeFailed {
		t.Fatalf("Code = %q", result.Code)
	}
	if _, statErr := os.Stat(cfg.DBPath); statErr == nil {
		t.Fatal("archive created without secret")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("Stat(db) = %v", statErr)
	}
}

func TestUnsupportedFixtureFailsClosed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "pocket", "unsupported")
	cfg := config.Config{
		Version:  1,
		DBPath:   filepath.Join(root, "pocketcrawl.db"),
		CacheDir: filepath.Join(root, "cache"),
		LogDir:   filepath.Join(root, "logs"),
	}
	result, err := sync.Run(context.Background(), cfg, sync.Options{
		FixtureDir:     fixtureDir,
		CrawlerVersion: "pocketcrawl-test",
		Secrets:        secret.MapProvider{},
	})
	if !sync.IsUnsupportedSchema(err) {
		t.Fatalf("Run() err = %v", err)
	}
	if result.Code != source.OutcomeUnsupportedSchema {
		t.Fatalf("Code = %q", result.Code)
	}
	if _, statErr := os.Stat(cfg.DBPath); statErr == nil {
		t.Fatal("archive created on unsupported schema")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("Stat(db) = %v", statErr)
	}
}

func mustRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
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
