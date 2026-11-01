package source_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	wsource "github.com/TheAngryPit/meetcrawl/internal/whisp/source"
	_ "modernc.org/sqlite"
)

func notesDB(t *testing.T, withDuration bool) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	extra := ""
	if withDuration {
		extra = ", audio_duration_seconds real"
	}
	if _, err := db.Exec(`create table notes (id integer primary key, content text, enhanced_content text, note_type text,
transcript text, created_at text, updated_at text, calendar_event_id text, participants text, deleted_at text` + extra + `);
insert into notes (id, note_type, created_at, updated_at, transcript) values (1, 'meeting', '2026-10-20 14:00:00', '2026-10-20 14:35:00', 'x');`); err != nil {
		t.Fatal(err)
	}
	if withDuration {
		if _, err := db.Exec(`update notes set audio_duration_seconds = 1800.5 where id = 1`); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestMeetingEndComesFromTheDurationWhenTheColumnExists(t *testing.T) {
	t.Parallel()
	notes, err := wsource.ReadMeetingNotes(context.Background(), notesDB(t, true))
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 20, 14, 30, 0, 500_000_000, time.UTC)
	if len(notes) != 1 || !notes[0].End.Equal(want) {
		t.Fatalf("End = %v, want %v", notes[0].End, want)
	}
}

func TestOlderOpenWhisprWithoutADurationColumnStillReads(t *testing.T) {
	t.Parallel()
	notes, err := wsource.ReadMeetingNotes(context.Background(), notesDB(t, false))
	if err != nil {
		t.Fatalf("ReadMeetingNotes() = %v, want it to read a database with no duration column", err)
	}
	if len(notes) != 1 || !notes[0].End.IsZero() {
		t.Fatalf("notes = %+v, want one note with no end", notes)
	}
}
