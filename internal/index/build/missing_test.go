package build_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TheAngryPit/meetcrawl/internal/index/build"
	"github.com/TheAngryPit/meetcrawl/internal/source"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	wsync "github.com/TheAngryPit/meetcrawl/internal/whisp/sync"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/testutil"
)

func TestRunSkipsArchivesThatDoNotExist(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	sourceDB := filepath.Join(root, "openwhispr", "transcriptions.db")
	if err := testutil.WriteSupportedDB(sourceDB); err != nil {
		t.Fatalf("WriteSupportedDB() = %v", err)
	}
	whispDB := filepath.Join(root, "whispcrawl.db")
	if _, err := wsync.Run(context.Background(), wconfig.Config{
		Version:  1,
		DBPath:   whispDB,
		CacheDir: filepath.Join(root, "cache"),
		LogDir:   filepath.Join(root, "logs"),
	}, wsync.Options{SourceDB: sourceDB, CrawlerVersion: "whispcrawl-test"}); err != nil {
		t.Fatalf("whisp sync: %v", err)
	}

	missing := filepath.Join(root, "never-synced", "gmeetcrawl.db")
	result, err := build.Run(context.Background(), build.Options{
		IndexDBPath: filepath.Join(root, "meetcrawl.db"),
		Sources: []build.SourceArchive{
			{Kind: source.KindOpenWhispr, Path: whispDB},
			{Kind: source.KindGMeetGemini, Path: missing},
		},
	})
	if err != nil {
		t.Fatalf("Run() with one unsynced source = %v, want success", err)
	}
	if result.Meetings == 0 {
		t.Fatal("Meetings = 0, want the synced source indexed")
	}
	if len(result.SkippedSources) != 1 || result.SkippedSources[0] != missing {
		t.Fatalf("SkippedSources = %v, want [%s]", result.SkippedSources, missing)
	}
}

func TestRunFailsWhenNoArchiveExists(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := build.Run(context.Background(), build.Options{
		IndexDBPath: filepath.Join(root, "meetcrawl.db"),
		Sources: []build.SourceArchive{
			{Kind: source.KindOpenWhispr, Path: filepath.Join(root, "nope.db")},
		},
	})
	if err == nil {
		t.Fatal("Run() with no archives = nil, want error")
	}
}
