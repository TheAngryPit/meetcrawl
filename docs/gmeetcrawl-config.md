# gmeetcrawl configuration

gmeetcrawl stores settings in `config.toml` under the crawlkit config path (see `gmeetcrawl init`).

## Meet folder roots

`meet_folder_roots` is a list of Drive folder ids or paths. Sync lists Gemini Docs only under these roots. The default is a single entry:

```toml
meet_folder_roots = ["Google Meet"]
```

You can add more than one root:

```toml
meet_folder_roots = ["Google Meet", "1AbCdEfGhIjKlMnOpQrStUvWxYz"]
```

Path entries are resolved under My Drive (`Google Meet`, `Archive/Meet`, and so on). Id entries are used directly.

The folder list is a scope filter. Ingest still requires a Gemini title match and a recognized export shape. A Doc may live directly under the root folder or inside a per-meeting subfolder.

## OAuth paths

`oauth_client_path` and `token_path` point at the desktop OAuth client JSON and the token store. Scopes stay read-only: `drive.readonly` and `calendar.events.readonly`.

## Archive paths

`db_path`, `cache_dir`, and `log_dir` follow crawlkit defaults for the platform. They are expanded from `~` on load.
