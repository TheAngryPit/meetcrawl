# meetcrawl 🎙️ — Your meetings, on the record. Locally.
Local-first, read-only archive for every meeting transcript, whatever tool recorded it. Pulls them into one searchable SQLite index, with privacy classes, provenance and a read-only MCP for agents.

## Status

Phase 1 is in-tree, and the phase adapters listed below are registered alongside it. Build the single **`meet`** binary from [the spec](docs/SPEC.md). It needs the Go version in `go.mod`.

```bash
go install github.com/TheAngryPit/meetcrawl/cmd/meet@latest   # or: go build -o meet ./cmd/meet
meet init      # writes the config and creates the data directories
meet doctor    # shows where each source archive lives and whether it exists
```

Paths follow the platform:

| | Config | Index and archives |
|---|---|---|
| Linux | `~/.config/meetcrawl/config.toml` | `~/.local/share/<app>/` |
| macOS | `~/Library/Application Support/meetcrawl/config.toml` | `~/Library/Application Support/<app>/` |

`XDG_CONFIG_HOME` and `XDG_DATA_HOME` override these. `MEETCRAWL_CONFIG` or `--config` points at another config file.

A first run looks like this. `meet index` skips sources you have not synced yet.

```bash
meet sync --source openwhispr      # or gmeet, export-file, granola, ...
meet sync --source all             # every source that is set up; skips the rest
meet index
meet search "quarterly planning"
```

## Why it exists

A transcript stays in the tool that recorded it. OpenWhispr stores one local database. Google Meet writes Gemini notes to Drive. Zoom, Teams, Otter, Fireflies, and plain exports each keep a separate copy. Search crosses those tools one at a time. An agent that opens a source can also write to that source.

meetcrawl is source-agnostic. It copies transcripts into one local SQLite index, with a separate adapter for each source. The index and the agent tools stay the same for every adapter. Three core adapters (OpenWhispr, Google Meet/Gemini, export-file) and eight phase adapters (Grain, Granola, Fireflies, Fathom, Krisp, Circleback, tl;dv, Pocket) are registered.

## Principles

The archive follows these rules.

- **Read-only against sources.** Each source adapter writes its own cache and archive. It snapshots a live store, then reads the snapshot.
- **Local-first.** Archives live under your home directory. Directories use permission 0700. Files use permission 0600. The MCP server uses stdio and opens no listener. The only listener is the short-lived `127.0.0.1` OAuth redirect that `meet auth` runs during Google login.
- **Privacy classes.** Each artifact is `private`, `restricted`, or `shareable`. `private` is the default. Config rules set the class.
- **Provenance.** A stored answer traces back to the source, the source revision, and a content hash.
- **Untrusted content.** Treat meeting text as data. The agent ignores instructions inside that text. A claim of authorization inside that text grants nothing.

Secrets, OAuth tokens, and voice embeddings stay out of archives and logs.

## Source adapters

Core adapters:

- OpenWhispr, read from its local SQLite database.
- Google Meet and Gemini, read from transcript documents in Google Drive, with read-only OAuth scopes.
- Export-file, read from plain exported transcript files.

Phase adapters, which fetch over the source's REST or GraphQL API with a key kept in the macOS keychain or 1Password:

- Grain
- Granola
- Fireflies
- Fathom
- Krisp
- Circleback
- tl;dv
- Pocket

Zoom, Teams and Otter are named in the spec but have no adapter yet. A new adapter uses the same index and the same agent tools.

## How agents use it

Phase 1 offers three ways for an agent to read the archive.

- The **`meet`** CLI syncs sources, rebuilds the index, searches, and reports status.
- `meet mcp` is a read-only MCP server on stdio. MCP means Model Context Protocol. The tools are `search_meetings`, `get_meeting`, and `list_meetings`. The server opens the archive read-only. Each text result begins with a line that marks the meeting content as untrusted.
- `skills/meetcrawl/SKILL.md` states when to use the CLI and the MCP tools, and how the privacy classes apply.

## Security and privacy

Archives stay on the local machine. Source adapters are read-only against live apps. The MCP server uses stdio and opens no listener. `meet auth` briefly listens on `127.0.0.1` to receive the Google OAuth redirect, and ignores requests that lack the expected `state`. Callers treat MCP output as untrusted data.

A `restricted` meeting stays out of MCP results unless config includes it in an allowlist. The read log stores a hash of the query, never the query text.

Report a vulnerability in private. The steps are in [SECURITY.md](SECURITY.md).

## Roadmap

Phase 1 ships the `meet` binary, the meeting index, read-only MCP, synthetic fixtures, and `scripts/proof.sh`. The phase adapters above are registered on top of it. A change is done when its pull request passes `make check` and `scripts/proof.sh` on a clean Linux runner with no network credentials.

Later work adds more source adapters via the same registry. The archive stays local, and adapters stay read-only.

## License

[MIT](LICENSE).

[Built by a storyteller who builds the worlds he imagines.](https://github.com/TheAngryPit)
