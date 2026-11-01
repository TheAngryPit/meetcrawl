package archive_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheAngryPit/meetcrawl/internal/index/archive"
	"github.com/TheAngryPit/meetcrawl/internal/source"
	_ "modernc.org/sqlite"
)

func TestMeetingExposesSourcesStartAndParticipants(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "meetcrawl.db")
	store, err := archive.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	meetings := []archive.MeetingRecord{{
		MeetingID: "m1", BestFidelity: source.FidelityTranscript, Privacy: source.PrivacyPrivate,
		Fidelities: []source.Fidelity{source.FidelityTranscript},
	}}
	contents := []archive.ContentRecord{{
		MeetingID: "m1", ContentHash: "h1", NormalizedText: "[00:00] Vitor: Hello.",
		Fidelity: source.FidelityTranscript, Language: "und",
		Sources: []archive.SourceLink{
			{
				Source: "openwhispr", SourceID: "7:transcript", SourceRevision: "2026-10-20 14:35:00",
				ProvenanceHash: "p1", Fidelity: source.FidelityTranscript,
				WindowStart: "2026-10-20T14:00:00Z", WindowEnd: "2026-10-20T14:30:00Z",
				Participants: `[{"name":"Ana","email":"ana@example.com"}]`,
			},
			{
				Source: "gmeet-gemini", SourceID: "doc-9", ProvenanceHash: "p2", Fidelity: source.FidelityTranscript,
				WindowStart: "2026-10-20T13:59:00Z", Participants: "Vitor, Ana",
			},
		},
	}}
	if err := store.ReplaceIndex(ctx, meetings, contents, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	ro, err := archive.OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()

	detail, err := ro.GetMeeting(ctx, "m1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if detail.StartUTC != "2026-10-20T13:59:00Z" {
		t.Fatalf("StartUTC = %q, want the earliest source start", detail.StartUTC)
	}
	got := detail.Contents[0].Sources
	if len(got) != 2 || got[0].Source != "gmeet-gemini" || got[1].Source != "openwhispr" {
		t.Fatalf("Sources = %+v, want both recorders ordered by source", got)
	}
	whispr := got[1]
	if whispr.SourceID != "7:transcript" || whispr.SourceRevision != "2026-10-20 14:35:00" ||
		whispr.StartUTC != "2026-10-20T14:00:00Z" || whispr.EndUTC != "2026-10-20T14:30:00Z" {
		t.Fatalf("openwhispr source = %+v", whispr)
	}
	if string(whispr.Participants) != `[{"name":"Ana","email":"ana@example.com"}]` {
		t.Fatalf("JSON participants = %s, want them passed through as JSON", whispr.Participants)
	}
	var plain string
	if err := json.Unmarshal(got[0].Participants, &plain); err != nil || plain != "Vitor, Ana" {
		t.Fatalf("plain participants = %s (%v), want a JSON string", got[0].Participants, err)
	}

	listed, err := ro.ListMeetings(ctx, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].StartUTC != "2026-10-20T13:59:00Z" {
		t.Fatalf("ListMeetings = %+v, want start_utc", listed)
	}
}

func TestSourcesWithoutWindowOrParticipantsStayAbsent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "meetcrawl.db")
	store, err := archive.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	err = store.ReplaceIndex(ctx,
		[]archive.MeetingRecord{{MeetingID: "m1", BestFidelity: source.FidelityNotes, Privacy: source.PrivacyPrivate}},
		[]archive.ContentRecord{{
			MeetingID: "m1", ContentHash: "h1", NormalizedText: "notes", Fidelity: source.FidelityNotes,
			Sources: []archive.SourceLink{{Source: "export-file", SourceID: "a", ProvenanceHash: "p", Fidelity: source.FidelityNotes}},
		}}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	ro, err := archive.OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	detail, err := ro.GetMeeting(ctx, "m1", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(detail)
	for _, key := range []string{`"start_utc"`, `"end_utc"`, `"participants"`} {
		if strings.Contains(string(raw), key) {
			t.Fatalf("%s present for a source that reports none: %s", key, raw)
		}
	}
}

const schemaV1ContentSources = `
create table meetings (meeting_id text primary key, best_fidelity text not null, privacy_class text not null, ical_uid text, event_start_utc text);
create table content_sources (
  meeting_id text not null, content_hash text not null, source text not null, source_id text not null,
  source_revision text not null default '', provenance_hash text not null, fidelity text not null,
  primary key (source, source_id));
create table schema_migrations (version integer not null);
insert into schema_migrations values (1);`

func TestOlderIndexStaysRefusedUntilItIsRebuilt(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "meetcrawl.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(schemaV1ContentSources); err != nil {
		t.Fatal(err)
	}
	db.Close()

	refused := func(when string) {
		t.Helper()
		if _, err := archive.OpenReadOnly(ctx, path); err == nil || !strings.Contains(err.Error(), "run meetcrawl index") {
			t.Fatalf("OpenReadOnly %s = %v, want a run meetcrawl index hint", when, err)
		}
	}
	refused("on a v1 index")

	// meetcrawl status and search open the store for writing. That must not
	// make the old rows look current.
	store, err := archive.Open(ctx, path)
	if err != nil {
		t.Fatalf("Open(v1 index) = %v", err)
	}
	refused("after a write open")

	if err := store.ReplaceIndex(ctx, nil, nil, time.Now()); err != nil {
		t.Fatalf("ReplaceIndex(v1 index) = %v, want the missing columns added", err)
	}
	store.Close()
	ro, err := archive.OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatalf("OpenReadOnly after the rebuild = %v", err)
	}
	ro.Close()
}
