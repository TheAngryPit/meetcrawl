package archive

const Schema = `
create table if not exists schema_migrations (
  version integer not null
);

create table if not exists artifacts (
  source_id text not null,
  fidelity text not null,
  source_revision text not null default '',
  privacy_class text not null,
  normalized_text text not null,
  content_hash text not null,
  provenance_hash text not null,
  window_start text,
  window_end text,
  language text not null default '',
  calendar_event_id text,
  participants text,
  updated_at text not null,
  primary key (source_id)
);

create index if not exists idx_artifacts_fidelity on artifacts(fidelity);

create virtual table if not exists artifacts_fts using fts5(
  normalized_text,
  content='artifacts',
  content_rowid='rowid',
  tokenize='unicode61 remove_diacritics 2'
);

create trigger if not exists artifacts_fts_insert after insert on artifacts begin
  insert into artifacts_fts(rowid, normalized_text) values (new.rowid, new.normalized_text);
end;

create trigger if not exists artifacts_fts_delete after delete on artifacts begin
  insert into artifacts_fts(artifacts_fts, rowid, normalized_text) values('delete', old.rowid, old.normalized_text);
end;

create trigger if not exists artifacts_fts_update after update on artifacts begin
  insert into artifacts_fts(artifacts_fts, rowid, normalized_text) values('delete', old.rowid, old.normalized_text);
  insert into artifacts_fts(rowid, normalized_text) values (new.rowid, new.normalized_text);
end;

create table if not exists sync_runs (
  id integer primary key autoincrement,
  code text not null,
  detail text,
  artifact_count integer not null default 0,
  finished_at text not null
);
`

const SchemaVersion = 1
