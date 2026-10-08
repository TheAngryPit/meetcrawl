package sync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	econfig "github.com/TheAngryPit/meetcrawl/internal/exportcrawl/config"
	"github.com/TheAngryPit/meetcrawl/internal/exportcrawl/sync"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func TestUnsupportedFixtureFailsClosed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "export-file", "unsupported")
	cfg := econfig.Config{
		Version:      1,
		DBPath:       filepath.Join(root, "exportcrawl.db"),
		CacheDir:     filepath.Join(root, "cache"),
		LogDir:       filepath.Join(root, "logs"),
		PrivacyClass: string(source.PrivacyPrivate),
	}
	result, err := sync.Run(context.Background(), cfg, sync.Options{FixtureDir: fixtureDir})
	if !sync.IsUnsupportedSchema(err) {
		t.Fatalf("Run() err = %v, want unsupported schema", err)
	}
	if result.Code != source.OutcomeUnsupportedSchema {
		t.Fatalf("Code = %q", result.Code)
	}
	if _, err := os.Stat(cfg.DBPath); err == nil {
		t.Fatalf("archive created on unsupported schema")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(db) = %v", err)
	}
}

func TestSyncSupportedFixture(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "export-file", "supported")
	cfg := econfig.Config{
		Version:      1,
		DBPath:       filepath.Join(root, "exportcrawl.db"),
		CacheDir:     filepath.Join(root, "cache"),
		LogDir:       filepath.Join(root, "logs"),
		PrivacyClass: string(source.PrivacyShareable),
	}
	before := dirSHA256(t, fixtureDir)
	result, err := sync.Run(context.Background(), cfg, sync.Options{
		FixtureDir:     fixtureDir,
		CrawlerVersion: "exportcrawl-test",
	})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if result.Code != source.OutcomeOK {
		t.Fatalf("Code = %q", result.Code)
	}
	if result.Artifacts != 4 {
		t.Fatalf("Artifacts = %d, want 4", result.Artifacts)
	}
	after := dirSHA256(t, fixtureDir)
	if before != after {
		t.Fatal("fixture dir hash changed after sync")
	}
}

func TestPrivacyClassFromConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "export-file", "supported")
	cfg := econfig.Config{
		Version:      1,
		DBPath:       filepath.Join(root, "exportcrawl.db"),
		CacheDir:     filepath.Join(root, "cache"),
		LogDir:       filepath.Join(root, "logs"),
		PrivacyClass: string(source.PrivacyShareable),
	}
	result, err := sync.Run(context.Background(), cfg, sync.Options{
		FixtureDir:     fixtureDir,
		CrawlerVersion: "exportcrawl-test",
	})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	for _, artifact := range result.Outcome.Artifacts {
		if artifact.Privacy != source.PrivacyShareable {
			t.Fatalf("Privacy = %q, want shareable from config", artifact.Privacy)
		}
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

func dirSHA256(t *testing.T, dir string) string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	var parts []byte
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, data...)
	}
	return source.ContentHash(string(parts))
}
