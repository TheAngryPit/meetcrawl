package archive

const Schema = `
create table if not exists schema_migrations (
  version integer not null
);

create table if not exists meetings (
  meeting_id text primary key,
  best_fidelity text not null,
  privacy_class text not null,
  ical_uid text,
  event_start_utc text
);

create table if not exists meeting_fidelities (
  meeting_id text not null,
  fidelity text not null,
  primary key (meeting_id, fidelity),
  foreign key (meeting_id) references meetings(meeting_id)
);

create table if not exists meeting_contents (
  meeting_id text not null,
  content_hash text not null,
  normalized_text text not null,
  fidelity text not null,
  language text not null default '',
  primary key (meeting_id, content_hash),
  foreign key (meeting_id) references meetings(meeting_id)
);

create table if not exists content_sources (
  meeting_id text not null,
  content_hash text not null,
  source text not null,
  source_id text not null,
  source_revision text not null default '',
  provenance_hash text not null,
  fidelity text not null,
  primary key (source, source_id),
  foreign key (meeting_id, content_hash) references meeting_contents(meeting_id, content_hash)
);

create index if not exists idx_content_sources_meeting on content_sources(meeting_id);

create virtual table if not exists meeting_contents_fts using fts5(
  normalized_text,
  content='meeting_contents',
  content_rowid='rowid',
  tokenize='unicode61 remove_diacritics 2'
);

create trigger if not exists meeting_contents_fts_insert after insert on meeting_contents begin
  insert into meeting_contents_fts(rowid, normalized_text) values (new.rowid, new.normalized_text);
end;

create trigger if not exists meeting_contents_fts_delete after delete on meeting_contents begin
  insert into meeting_contents_fts(meeting_contents_fts, rowid, normalized_text) values('delete', old.rowid, old.normalized_text);
end;

create trigger if not exists meeting_contents_fts_update after update on meeting_contents begin
  insert into meeting_contents_fts(meeting_contents_fts, rowid, normalized_text) values('delete', old.rowid, old.normalized_text);
  insert into meeting_contents_fts(rowid, normalized_text) values (new.rowid, new.normalized_text);
end;

create table if not exists index_runs (
  id integer primary key autoincrement,
  meeting_count integer not null,
  content_count integer not null,
  source_link_count integer not null,
  finished_at text not null
);
`

const SchemaVersion = 1
