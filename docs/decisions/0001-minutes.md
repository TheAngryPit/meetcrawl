# 0001. Minutes gate

Phase 0 for [docs/SPEC.md](../SPEC.md) section 8. Read on 2026-10-08 at [`silverstein/minutes` `109fbe16`](https://github.com/silverstein/minutes/commit/109fbe16ab16b6a7a6cd2e2de364bd437a64010a).

## Recommendation

Build meetcrawl.

Minutes is a local capture app. It transcribes on the device, stores markdown under `~/meetings/`, and can pull in a Granola export or a folder of `.md`, `.markdown`, and `.txt` files. Those imports become Minutes meeting files. The public issues and pull requests show that work, and they show no OpenWhispr importer and no Google Meet/Gemini transcript importer. The shipped MCP server also records, processes audio, and resummarizes, so an upstream importer would still sit inside a different contract than a read-only, per-source archive with privacy classes and provenance hashes.

## Activity

Source: the GitHub repository API for `silverstein/minutes` on 2026-10-08.

| | |
|---|---|
| Created | 2026-03-18 |
| Last push | 2026-10-08T05:08:14Z |
| Commits on `main` | 2,326 |
| Commits since 2026-09-08 | 197 |
| Releases | 90 GitHub releases. Newest published release and newest tag: [v0.28.1](https://github.com/silverstein/minutes/releases/tag/v0.28.1) (2026-10-07) |
| Issues | 59 open, 197 closed |
| Pull requests | 18 open, 708 merged, 113 closed without merge |
| Other | MIT, Rust, 1,539 stars, 168 forks, not archived |

A later commit message says "Release Minutes 0.28.2". There is no `v0.28.2` tag and no GitHub release with that name. Whether a 0.28.2 package exists on crates.io or npm is unverified.

## What I read

File links use commit `109fbe16`.

- [README](https://github.com/silverstein/minutes/blob/109fbe16/README.md). Record and transcribe locally. Markdown in `~/meetings/`. MCP for agents. Text leaves the machine when a chosen cloud summarizer runs. The CLI section includes import. The comparison table is Granola, Otter.ai, Anarlog, and Minutes.
- [LICENSE](https://github.com/silverstein/minutes/blob/109fbe16/LICENSE). MIT. Copyright 2026 Mat Silverstein.
- [CONTRIBUTING.md](https://github.com/silverstein/minutes/blob/109fbe16/CONTRIBUTING.md). Solo project. Named help areas are Windows/Linux testing, new read-only SDK tools, CLI query commands, and docs. The TypeScript SDK is described as read-only. No section on source importers.
- [docs/architecture/README.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/architecture/README.md). Pipeline is audio, local transcription, diarization, optional LLM summary, markdown. The MCP list includes `list_meetings`, `search_meetings`, and `get_meeting`, and also `start_recording`, `stop_recording`, `process_audio`, live transcript, and copilot.
- [docs/architecture/frontmatter-schema.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/architecture/frontmatter-schema.md). Interop contract for meeting files. `source` is a free string (`live-recording`, `voice-memo`, `upload` are the examples). `sensitivity: restricted` is an agent-layer exclusion. `visibility` is `private` or `team`, and the page says Minutes does not enforce ACLs. The page does not define `imported_from` or a provenance hash.
- [docs/architecture/consent-enforcement.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/architecture/consent-enforcement.md). Contract table and override logging: restricted meetings stay out of agent search, MCP recall, and ingest unless a logged override is set. I read that section, not the rest of the file.
- [docs/architecture/multi-source-capture.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/architecture/multi-source-capture.md). Status, problem, product contract, and architecture. "Source" means microphone and system-audio stems for call capture, extending [issue #34](https://github.com/silverstein/minutes/issues/34).
- [docs/architecture/context-store.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/architecture/context-store.md). Opening contract: `~/meetings/*.md` is the meeting record; `~/.minutes/context.db` is a desktop-context sidecar.
- [docs/features.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/features.md). Record, process audio files, search, live transcript. Browser Meet and Teams detection is marked experimental.
- [docs/switching-from-granola.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/switching-from-granola.md). `minutes import granola` reads `~/.granola-archivist/output/`. `minutes import text --dir` imports `.md`, `.markdown`, and `.txt`. A linked companion tool uses Granola's unofficial API.
- [docs/integration/agent-integrations.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/integration/agent-integrations.md). How to add an agent host. "Gemini CLI" on this page is a client that can call Minutes, not a Meet transcript source.
- [docs/plans/minutes-archive-discovery-2026-07-30.md](https://github.com/silverstein/minutes/blob/109fbe16/docs/plans/minutes-archive-discovery-2026-07-30.md). Recommendation and shipped-facts sections. A separate Archive app inventories documents (PDF, DOCX, text), keeps originals read-only, and stores hashes on sidecars. That is a document vault.
- [.agents/skills/minutes/minutes-ingest/SKILL.md](https://github.com/silverstein/minutes/blob/109fbe16/.agents/skills/minutes/minutes-ingest/SKILL.md). `minutes ingest` copies frontmatter facts into a wiki. Provenance there is "which meeting this fact came from."
- [.agents/skills/minutes/minutes-mcp-recall/references/tool-map.md](https://github.com/silverstein/minutes/blob/109fbe16/.agents/skills/minutes/minutes-mcp-recall/references/tool-map.md). 34 MCP tools, including `start_recording`, `stop_recording`, `process_audio`, `start_dictation`, `start_live_transcript`, `ingest_meeting`, `resummarize_meeting`, and `add_note`, plus the recall tools.
- [crates/cli/src/main.rs](https://github.com/silverstein/minutes/blob/109fbe16/crates/cli/src/main.rs). The `Import` command and the `match from` dispatch. Accepted sources are `granola`, `text`, or an existing directory routed to text import. Anything else errors and names those sources. An audio path is a compatibility alias to `minutes process`. Text import writes `source: text-import` and `imported_from`. Granola import writes `source: granola-import`.

Issues and pull requests:

- [Issue #515](https://github.com/silverstein/minutes/issues/515), closed by [PR #516](https://github.com/silverstein/minutes/pull/516) (merged 2026-07-21, author `silverstein`). Generic text-archive importer. The issue and the pull request landed the same day.
- [PR #319](https://github.com/silverstein/minutes/pull/319) (merged 2026-06-12). Docs for how the Granola export directory gets filled. It points at open [issue #318](https://github.com/silverstein/minutes/issues/318) (updated 2026-09-15): import Granola through its unofficial API inside Minutes.
- [PR #80](https://github.com/silverstein/minutes/pull/80) (merged 2026-04-06, author `calvindotsg`). Docs that leave `granola-to-minutes` as a companion tool. The body cites #78. That number now returns 404. What #78 said is unverified.
- [Issue #34](https://github.com/silverstein/minutes/issues/34). Linux audio device selection. Closed. This is the capture issue behind the multi-source doc.
- Meet search hits are call detection, for example [PR #53](https://github.com/silverstein/minutes/pull/53) and [PR #120](https://github.com/silverstein/minutes/pull/120). I used the titles. I did not read those diffs.
- Issue search for `openwhispr`: no results. Pull-request search for `importer`: #516 and #319, plus an unrelated dependency pull request. A checkout grep of `109fbe16` for `OpenWhispr` and `Notes by Gemini` returned no hits.

## Fit

| Principle | Minutes at this commit |
|---|---|
| Read-only sources | `minutes import` reads a local Granola export or a text directory and writes new meeting files. The app also records audio. Issue #318 proposes calling Granola's unofficial API. |
| Local-first | Meeting files are local markdown with mode `0600`. Cloud summarization runs when the user selects it. |
| Privacy classes | `sensitivity: restricted` is kept off agent surfaces unless a logged override is on. `visibility` is a `private` or `team` label. There is no `shareable` class and no calendar, folder, or title rule that assigns one. |
| Provenance | `source` and, for text import, `imported_from` say how the file was written. Consent fields record a disclosure basis. A meeting answer is not bound to a hash of source id, source revision, content hash, and crawler version. |
| Read-only MCP | Recall tools are on the same server as recording, audio processing, dictation, ingest, and resummarize. CONTRIBUTING asks for more read-only query tools in the SDK. |
| One adapter per source | Shipped import sources are Granola and generic text. Text import folds `.md`, `.markdown`, and `.txt` into one markdown schema. There is no OpenWhispr SQLite adapter and no Drive or Gemini Docs adapter. Multi-source capture is two audio stems, not two transcript providers. |

## Unverified

- Whether a maintainer would merge an OpenWhispr importer or a Meet/Gemini transcript importer. The public issues and pull requests I searched do not say yes or no.
- The body of every open issue. I searched titles and read the import threads above. The open issue count is 59; I listed 30 titles.
- crates.io and npm for a 0.28.2 package.
- Runtime behavior. I did not install or run Minutes.
- The deleted or missing #78 cited by PR #80.
