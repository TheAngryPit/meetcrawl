package build_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	calprovider "github.com/TheAngryPit/meetcrawl/internal/calendar/provider"
	"github.com/TheAngryPit/meetcrawl/internal/index/archive"
	"github.com/TheAngryPit/meetcrawl/internal/index/build"
	"github.com/TheAngryPit/meetcrawl/internal/source"
	wconfig "github.com/TheAngryPit/meetcrawl/internal/whisp/config"
	wsync "github.com/TheAngryPit/meetcrawl/internal/whisp/sync"
	"github.com/TheAngryPit/meetcrawl/internal/whisp/testutil"
	_ "modernc.org/sqlite"
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

func TestSourceMetadataKeepsWhatTheRecorderReportedNotTheCalendarEvent(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	repoRoot := mustRepoRoot(t)
	sourceDB := filepath.Join(root, "openwhispr", "transcriptions.db")
	if err := testutil.WriteSupportedDB(sourceDB); err != nil {
		t.Fatalf("WriteSupportedDB() = %v", err)
	}
	// The calendar fixture event runs 14:00 to 14:30 with 2 attendees and
	// matches this note by calendar_event_id. The recorder saw 14:05 to 14:25
	// and reported no participants.
	raw, err := sql.Open("sqlite", sourceDB)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`alter table notes add column audio_duration_seconds real;
update notes set created_at = '2026-01-15T14:05:00Z', audio_duration_seconds = 1200, participants = null
where note_type = 'meeting';`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	whispDB := filepath.Join(root, "whispcrawl.db")
	if _, err := wsync.Run(context.Background(), wconfig.Config{
		Version: 1, DBPath: whispDB, CacheDir: filepath.Join(root, "cache"), LogDir: filepath.Join(root, "logs"),
	}, wsync.Options{SourceDB: sourceDB, CrawlerVersion: "whispcrawl-test"}); err != nil {
		t.Fatalf("whisp sync: %v", err)
	}
	indexDB := filepath.Join(root, "meetcrawl.db")
	if _, err := build.Run(context.Background(), build.Options{
		IndexDBPath: indexDB,
		Calendar:    calprovider.Options{FixtureDir: filepath.Join(repoRoot, "testdata", "fixtures", "calendar", "synthetic")},
		Sources:     []build.SourceArchive{{Kind: source.KindOpenWhispr, Path: whispDB}},
	}); err != nil {
		t.Fatalf("index build: %v", err)
	}

	ro, err := archive.OpenReadOnly(context.Background(), indexDB)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	listed, err := ro.ListMeetings(context.Background(), 10, nil)
	if err != nil || len(listed) != 1 {
		t.Fatalf("ListMeetings = %+v, %v", listed, err)
	}
	detail, err := ro.GetMeeting(context.Background(), listed[0].MeetingID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ICalUID != "cal-synthetic-001" || detail.EventStartUTC != "2026-01-15T14:00:00Z" {
		t.Fatalf("calendar linkage = %q at %q, want the matched event", detail.ICalUID, detail.EventStartUTC)
	}
	for _, content := range detail.Contents {
		for _, src := range content.Sources {
			if src.StartUTC != "2026-01-15T14:05:00Z" || src.EndUTC != "2026-01-15T14:25:00Z" {
				t.Fatalf("source window = %s to %s, want the recorder's 14:05 to 14:25 and not the event's", src.StartUTC, src.EndUTC)
			}
			if len(src.Participants) != 0 {
				t.Fatalf("participants = %s, want none: the recorder reported none and the event's attendee count is not a roster", src.Participants)
			}
		}
	}
	if detail.StartUTC != "2026-01-15T14:05:00Z" {
		t.Fatalf("meeting start_utc = %q, want the recorder's start", detail.StartUTC)
	}
}
