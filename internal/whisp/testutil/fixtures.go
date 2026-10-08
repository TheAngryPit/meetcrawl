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
