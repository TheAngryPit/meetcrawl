# Phase 1 headless proof (SPEC section 9)

When Phase 1 code lands, product done means `make check && scripts/proof.sh` on a clean Linux runner without network credentials, covering all seven checks in docs/SPEC.md section 9.

## Sub-features

- `proof-not-runnable` — Doctor reports not runnable when `go.mod` or `scripts/proof.sh` is missing.
- `proof-sync-ingest` — whispcrawl and gmeetcrawl fixture sync counts (via `scripts/proof.sh`).
- `proof-index-dedup` — index meetings, dedup, adhoc, pt-PT FTS (via `scripts/proof.sh`).
- `proof-mcp-readlog` — MCP stdio, untrusted prefix, restricted hidden, read_log (via `scripts/proof.sh`).
- `proof-readonly-rebuild` — source sha256 stable, rebuildable index hash (via `scripts/proof.sh`).
- `proof-metadata-deps` — meetcrawl `metadata --json` / crawlbar manifest (not in proof script until implemented).
- `proof-unsupported-schema` — whispcrawl and gmeetcrawl unknown schema fail closed (via `scripts/proof.sh`).

## How to get to it (user POV)

- After Phase 1 ships, run the proof from the repo root on Linux with fixtures only (no live credentials).

## Driving it with meetcrawl-verify

Preconditions:

- Both `go.mod` and `scripts/proof.sh` exist at repo root.

- When `go.mod` and `scripts/proof.sh` exist, run `"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" drive --feature phase1-headless-proof`. Expect exit **0** after `make check && scripts/proof.sh`.
- When either file is missing, the same command exits **2** with message that proof is not runnable.
- Proof artifacts go under gitignored `proof/` while verification evidence stays in `$EVIDENCE_DIR`.

## Gotchas

- Do not implement or commit `scripts/proof.sh` from this skill; only drive it once it exists on the branch under test.
- Evidence for PR quotes must live in `$EVIDENCE_DIR`, not only under gitignored `proof/`.
