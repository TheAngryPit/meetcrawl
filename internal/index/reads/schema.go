package reads

const Schema = `
create table if not exists schema_migrations (
  version integer not null
);

create table if not exists read_log (
  id integer primary key autoincrement,
  ts text not null,
  surface text not null,
  client_name text,
  tool text not null,
  meeting_id text,
  artifact_id text,
  query_hash text not null
);

create index if not exists idx_read_log_ts on read_log(ts);
`

const SchemaVersion = 1
