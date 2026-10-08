package sync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/source"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/sync"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/testutil"
)

func TestUnsupportedSchemaFailsClosed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourceDB := filepath.Join(root, "unsupported", "transcriptions.db")
	if err := testutil.WriteUnsupportedDB(sourceDB); err != nil {
		t.Fatalf("WriteUnsupportedDB() = %v", err)
	}
	cfg := config.Config{
		Version:  1,
		DBPath:   filepath.Join(root, "whispcrawl.db"),
		CacheDir: filepath.Join(root, "cache"),
		LogDir:   filepath.Join(root, "logs"),
	}
	result, err := sync.Run(context.Background(), cfg, sync.Options{SourceDB: sourceDB})
	if !sync.IsUnsupportedSchema(err) {
		t.Fatalf("Run() err = %v, want unsupported schema", err)
	}
	if result.Code != source.OutcomeUnsupportedSchema {
		t.Fatalf("Code = %q, want unsupported_schema", result.Code)
	}
	outcome := source.SyncOutcome{Code: result.Code, Detail: result.Detail}
	if err := outcome.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
	if _, err := os.Stat(cfg.DBPath); err == nil {
		t.Fatalf("archive db created on unsupported schema: %s", cfg.DBPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(archive) = %v", err)
	}
}

func TestSyncSupportedFixture(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourceDB := filepath.Join(root, "source", "transcriptions.db")
	if err := testutil.WriteSupportedDB(sourceDB); err != nil {
		t.Fatalf("WriteSupportedDB() = %v", err)
	}
	cfg := config.Config{
		Version:  1,
		DBPath:   filepath.Join(root, "whispcrawl.db"),
		CacheDir: filepath.Join(root, "cache"),
		LogDir:   filepath.Join(root, "logs"),
	}
	before := fileSHA256(t, sourceDB)
	result, err := sync.Run(context.Background(), cfg, sync.Options{SourceDB: sourceDB, CrawlerVersion: "whispcrawl-test"})
	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if result.Code != source.OutcomeOK {
		t.Fatalf("Code = %q", result.Code)
	}
	if result.Artifacts != 3 {
		t.Fatalf("Artifacts = %d, want 3 (transcript, notes, summary)", result.Artifacts)
	}
	after := fileSHA256(t, sourceDB)
	if before != after {
		t.Fatal("source db hash changed after sync")
	}
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) = %v", path, err)
	}
	sum := source.ContentHash(string(data))
	return sum
}
