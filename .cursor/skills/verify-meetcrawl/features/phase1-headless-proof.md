# Phase 1 headless proof (SPEC section 9)

When Phase 1 code lands, product done means `make check && scripts/proof.sh` on a clean Linux runner without network credentials, covering all seven checks in docs/SPEC.md section 9.

## Sub-features

- `proof-not-runnable` — today: missing `go.mod` and/or `scripts/proof.sh` (Doctor reports not runnable).
- `proof-sync-ingest` — whispcrawl and gmeetcrawl fixture sync counts (not drivable yet).
- `proof-index-dedup` — index meetings, dedup, adhoc, pt-PT FTS (not drivable yet).
- `proof-mcp-readlog` — MCP stdio, untrusted prefix, restricted hidden, read_log (not drivable yet).
- `proof-readonly-rebuild` — source sha256 stable, rebuildable index hash (not drivable yet).
- `proof-metadata-deps` — crawlkit.control.v1 metadata, no crawlkit/remote dep (not drivable yet).
- `proof-unsupported-schema` — whispcrawl unknown schema fails closed (not drivable yet).

## How to get to it (user POV)

- After Phase 1 ships, run the proof from the repo root on Linux with fixtures only (no live credentials).

## Driving it with meetcrawl-verify

Preconditions:

- Both `go.mod` and `scripts/proof.sh` exist at repo root.

- **Today.** Run `"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" drive --feature phase1-headless-proof`. Expect exit code **2** and message that proof is not runnable.
- **When runnable.** Same command runs `make check && scripts/proof.sh`; proof artifacts go under gitignored `proof/` while verification evidence stays in `$EVIDENCE_DIR`.

## Gotchas

- Do not implement or commit `scripts/proof.sh` from this skill; only drive it once it exists on the branch under test.
- Evidence for PR quotes must live in `$EVIDENCE_DIR`, not only under gitignored `proof/`.
