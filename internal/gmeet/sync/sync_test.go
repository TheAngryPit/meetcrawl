package sync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/sync"
	"github.com/TheAngryPit/meetcrawl/internal/source"
)

func TestUnsupportedFixtureFailsClosed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "gdrive", "unsupported")
	cfg := config.Config{
		Version:  1,
		DBPath:   filepath.Join(root, "gmeetcrawl.db"),
		CacheDir: filepath.Join(root, "cache"),
		LogDir:   filepath.Join(root, "logs"),
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

func TestSyncCustomFolderRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	fixtureDir := filepath.Join(root, "custom")
	if err := os.MkdirAll(filepath.Join(fixtureDir, "exports"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "version": 1,
  "drive_files": [{
    "id": "gdrive-custom-001",
    "name": "Synthetic archive - Notes by Gemini",
    "modifiedTime": "2026-02-10T11:00:00Z",
    "revisionId": "rev-custom-001",
    "parentPath": "Synthetic Archive",
    "exportMarkdownPath": "exports/custom.md"
  }],
  "calendar_events": []
}`
	exportMD := `# **Notes by Gemini**

### Summary
Custom folder blorp.`
	if err := os.WriteFile(filepath.Join(fixtureDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "exports", "custom.md"), []byte(exportMD), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		Version:         1,
		DBPath:          filepath.Join(root, "gmeetcrawl.db"),
		CacheDir:        filepath.Join(root, "cache"),
		LogDir:          filepath.Join(root, "logs"),
		MeetFolderRoots: []string{"Synthetic Archive"},
	}
	result, err := sync.Run(context.Background(), cfg, sync.Options{
		FixtureDir:     fixtureDir,
		CrawlerVersion: "gmeetcrawl-test",
	})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if result.Artifacts != 1 {
		t.Fatalf("Artifacts = %d, want 1", result.Artifacts)
	}
}

func TestSyncSupportedFixture(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "gdrive", "supported")
	cfg := config.Config{
		Version:  1,
		DBPath:   filepath.Join(root, "gmeetcrawl.db"),
		CacheDir: filepath.Join(root, "cache"),
		LogDir:   filepath.Join(root, "logs"),
	}
	before := dirSHA256(t, fixtureDir)
	result, err := sync.Run(context.Background(), cfg, sync.Options{
		FixtureDir:     fixtureDir,
		CrawlerVersion: "gmeetcrawl-test",
	})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if result.Code != source.OutcomeOK {
		t.Fatalf("Code = %q", result.Code)
	}
	if result.Artifacts != 3 {
		t.Fatalf("Artifacts = %d, want 3", result.Artifacts)
	}
	after := dirSHA256(t, fixtureDir)
	if before != after {
		t.Fatal("fixture dir hash changed after sync")
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
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		paths = append(paths, path)
		return nil
	})
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
