package build_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	calprovider "github.com/TheAngryPit/meetcrawl/internal/calendar/provider"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/config"
	"github.com/TheAngryPit/meetcrawl/internal/gmeet/sync"
	"github.com/TheAngryPit/meetcrawl/internal/index/archive"
	"github.com/TheAngryPit/meetcrawl/internal/index/build"
	"github.com/TheAngryPit/meetcrawl/internal/source"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	wsync "github.com/TheAngryPit/meetcrawl/internal/whisp/sync"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/testutil"
)

func TestIndexDedupSearchAndRebuild(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)

	whispDB := filepath.Join(root, "whispcrawl.db")
	sourceDB := filepath.Join(root, "openwhispr", "transcriptions.db")
	if err := testutil.WriteSupportedDB(sourceDB); err != nil {
		t.Fatalf("WriteSupportedDB() = %v", err)
	}
	whispCfg := wconfig.Config{
		Version:  1,
		DBPath:   whispDB,
		CacheDir: filepath.Join(root, "whisp-cache"),
		LogDir:   filepath.Join(root, "whisp-logs"),
	}
	if _, err := wsync.Run(context.Background(), whispCfg, wsync.Options{
		SourceDB:       sourceDB,
		CrawlerVersion: "whispcrawl-test",
	}); err != nil {
		t.Fatalf("whisp sync: %v", err)
	}

	gmeetDB := filepath.Join(root, "gmeetcrawl.db")
	fixtureDir := filepath.Join(repoRoot, "testdata", "fixtures", "gdrive", "supported")
	gmeetCfg := config.Config{
		Version:  1,
		DBPath:   gmeetDB,
		CacheDir: filepath.Join(root, "gmeet-cache"),
		LogDir:   filepath.Join(root, "gmeet-logs"),
	}
	if _, err := sync.Run(context.Background(), gmeetCfg, sync.Options{
		FixtureDir:     fixtureDir,
		CrawlerVersion: "gmeetcrawl-test",
	}); err != nil {
		t.Fatalf("gmeet sync: %v", err)
	}

	indexDB := filepath.Join(root, "meetcrawl.db")
	calendarFix := filepath.Join(repoRoot, "testdata", "fixtures", "calendar", "synthetic")
	result, err := build.Run(context.Background(), build.Options{
		IndexDBPath: indexDB,
		Calendar: calprovider.Options{
			FixtureDir: calendarFix,
		},
		Sources: []build.SourceArchive{
			{Kind: source.KindOpenWhispr, Path: whispDB},
			{Kind: source.KindGMeetGemini, Path: gmeetDB},
		},
	})
	if err != nil {
		t.Fatalf("index build: %v", err)
	}
	if result.Meetings != 2 {
		t.Fatalf("Meetings = %d, want 2 (one calendar deduped, one adhoc)", result.Meetings)
	}

	expectedCalendarID := calendarMeetingID(t, "cal-synthetic-001", "2026-01-15T14:00:00Z")
	store, err := archive.Open(context.Background(), indexDB)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer store.Close()

	var bestFidelity string
	var sourceCount int
	err = store.DB().QueryRowContext(context.Background(), `
select m.best_fidelity, count(cs.source_id)
from meetings m
join meeting_contents c on c.meeting_id = m.meeting_id
join content_sources cs on cs.meeting_id = c.meeting_id and cs.content_hash = c.content_hash
where m.meeting_id = ?
  and c.normalized_text like '%Synthetic standup transcript%'
group by m.meeting_id
`, expectedCalendarID).Scan(&bestFidelity, &sourceCount)
	if err != nil {
		t.Fatalf("query deduped transcript: %v", err)
	}
	if bestFidelity != "transcript" {
		t.Fatalf("best_fidelity = %q, want transcript", bestFidelity)
	}
	if sourceCount != 2 {
		t.Fatalf("source links on deduped transcript = %d, want 2", sourceCount)
	}

	var adhocCount int
	if err := store.DB().QueryRowContext(context.Background(), `
select count(*) from meetings where meeting_id like 'adhoc:%'
`).Scan(&adhocCount); err != nil {
		t.Fatalf("count adhoc: %v", err)
	}
	if adhocCount != 1 {
		t.Fatalf("adhoc meetings = %d, want 1", adhocCount)
	}

	hits, err := store.Search(context.Background(), "reuniao", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected pt-PT query without accents to match accented fixture text")
	}
	if !strings.Contains(strings.ToLower(hits[0].Snippet), "reuni") {
		t.Fatalf("snippet = %q, want accented reunião match", hits[0].Snippet)
	}

	firstDump, err := store.OrderedRowDump(context.Background())
	if err != nil {
		t.Fatalf("dump: %v", err)
	}
	firstHash := source.ContentHash(firstDump)

	if err := os.Remove(indexDB); err != nil {
		t.Fatalf("remove index db: %v", err)
	}
	if _, err := build.Run(context.Background(), build.Options{
		IndexDBPath: indexDB,
		Calendar: calprovider.Options{
			FixtureDir: calendarFix,
		},
		Sources: []build.SourceArchive{
			{Kind: source.KindOpenWhispr, Path: whispDB},
			{Kind: source.KindGMeetGemini, Path: gmeetDB},
		},
	}); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	store2, err := archive.Open(context.Background(), indexDB)
	if err != nil {
		t.Fatalf("reopen index: %v", err)
	}
	defer store2.Close()
	secondDump, err := store2.OrderedRowDump(context.Background())
	if err != nil {
		t.Fatalf("second dump: %v", err)
	}
	secondHash := source.ContentHash(secondDump)
	if firstHash != secondHash {
		t.Fatalf("rebuild hash mismatch:\nfirst=%s\nsecond=%s", firstHash, secondHash)
	}
}

func calendarMeetingID(t *testing.T, icalUID, start string) string {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, start)
	if err != nil {
		t.Fatal(err)
	}
	payload := icalUID + "|" + ts.UTC().Format(time.RFC3339)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
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
