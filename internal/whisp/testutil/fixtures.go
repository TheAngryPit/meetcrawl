package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TheAngryPit/meetcrawl/internal/whisp/schema"
	_ "modernc.org/sqlite"
)

func WriteSupportedDB(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	_ = os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `
create table notes (
  id integer primary key autoincrement,
  content text not null default '',
  enhanced_content text,
  note_type text not null default 'personal',
  transcript text,
  created_at text not null default '2026-01-15T14:00:00Z',
  updated_at text not null default '2026-01-15T14:30:00Z',
  calendar_event_id text,
  participants text,
  deleted_at text
);
insert into notes (
  content, enhanced_content, note_type, transcript, calendar_event_id, participants
) values
  ('Synthetic standup notes body.', 'Synthetic standup summary.', 'meeting',
   'Synthetic standup transcript for fixture ingest.', 'cal-synthetic-001', 'speaker-a, speaker-b'),
  ('Personal scratch pad.', '', 'personal', '', null, null);
`); err != nil {
		return err
	}
	return schema.ValidateNotesTable(context.Background(), db)
}

// WriteIOSSupportedDB writes an OpenWhispr mobile-layout SQLite DB (schema.ts notes table).
func WriteIOSSupportedDB(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	_ = os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(context.Background(), `
create table notes (
  id integer primary key autoincrement,
  title text not null default 'Untitled',
  content text not null default '',
  folder_id integer,
  note_type text default 'personal',
  source_file text,
  audio_duration_seconds real,
  enhanced_content text,
  enhancement_prompt text,
  enhanced_at_content_hash text,
  transcript text,
  diarization_enabled integer not null default 0,
  expected_speaker_count integer,
  transcription_status text default 'idle',
  calendar_event_id text,
  participants text,
  client_note_id text,
  remote_id text,
  deleted_at text,
  pending_sync integer not null default 0,
  is_private integer not null default 0,
  space_id integer,
  owner_user_id text,
  updated_by_user_id text,
  cloud_updated_at text,
  conflict_server_note text,
  left_team integer not null default 0,
  created_at text default '2026-01-15T14:00:00Z',
  updated_at text default '2026-01-15T14:30:00Z'
);
insert into notes (
  title, content, enhanced_content, note_type, transcript,
  calendar_event_id, participants, client_note_id, remote_id, deleted_at
) values
  ('Synthetic ios meeting title', 'Synthetic ios meeting notes body.', 'Synthetic ios meeting summary.',
   'meeting', '{"segments":[{"text":"Synthetic ios transcript for fixture ingest."}]}',
   'cal-synthetic-001', 'participant-alpha, participant-beta', 'client-note-synthetic-001', 'remote-synthetic-001', null),
  ('Deleted ios meeting', 'Should not ingest.', '', 'meeting', 'Synthetic deleted row transcript.',
   null, null, 'client-note-deleted-001', 'remote-deleted-001', '2026-01-15T15:00:00Z'),
  ('Personal ios note', 'Non-meeting scratch.', '', 'personal', '', null, null, 'client-note-personal-001', null, null);
`); err != nil {
		return err
	}
	return schema.ValidateNotesTable(context.Background(), db)
}

func WriteUnsupportedDB(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	_ = os.Remove(path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	// Legacy layout without transcript column (supported ingest must fail closed).
	if _, err := db.ExecContext(context.Background(), `
create table notes (
  id integer primary key autoincrement,
  content text not null default '',
  enhanced_content text,
  note_type text not null default 'personal',
  created_at text not null default '2026-01-15T14:00:00Z',
  updated_at text not null default '2026-01-15T14:30:00Z',
  calendar_event_id text,
  participants text,
  deleted_at text
);
insert into notes (content, note_type) values ('Legacy meeting without transcript column.', 'meeting');
`); err != nil {
		return err
	}
	if err := schema.ValidateNotesTable(context.Background(), db); err == nil {
		return fmt.Errorf("unsupported fixture unexpectedly passed schema validation")
	}
	return nil
}
