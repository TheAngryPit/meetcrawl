# Google Meet / Gemini (`gmeet` adapter)

Configure the gmeet source through the unified **`meet`** config at `~/.config/meetcrawl/config.toml` (see `meet init`). Legacy standalone `gmeetcrawl` configs are not imported automatically.

## Archive path

The gmeet SQLite archive path is the top-level key **`gmeetcrawl_db`** (default under `~/.local/share/gmeetcrawl/`). Per-source settings live under **`[gmeet]`**.

## Meet folder roots

`meet_folder_roots` is a list of Drive folder ids or paths. Sync lists Gemini Docs only under these roots. The default is a single entry:

```toml
[gmeet]
meet_folder_roots = ["Google Meet"]
```

You can add more than one root:

```toml
[gmeet]
meet_folder_roots = ["Google Meet", "1AbCdEfGhIjKlMnOpQrStUvWxYz"]
```

Path entries are resolved under My Drive (`Google Meet`, `Archive/Meet`, and so on). Id entries are used directly.

The folder list is a scope filter. Ingest still requires a Gemini title match and a recognized export shape. A Doc may live directly under the root folder or inside a per-meeting subfolder.

## OAuth paths

Under `[gmeet]`, `oauth_client_path` and `token_path` point at the desktop OAuth client JSON and the token store. Run **`meet auth`** (optional `--client <path>`, `--manual`). Scopes stay read-only: `drive.readonly` and `calendar.events.readonly`.

Calendar enrichment at **`meet index`** can reuse the same token paths via `[calendar]` or `[gmeet]`.

## Cache and logs

`[gmeet]` also carries `cache_dir` and `log_dir` (crawlkit-style paths, expanded from `~` on load).
