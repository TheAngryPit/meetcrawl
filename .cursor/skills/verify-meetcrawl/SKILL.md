---
name: verify-meetcrawl
description: Verify meetcrawl repo health today (required docs, CI wiring, secrets/forbidden paths, SPEC integrity). When Phase 1 lands, drive headless `make check && scripts/proof.sh` per docs/SPEC.md section 9. Use before shipping doc/CI changes or when proving the repo matches the spec contract.
---

# verify-meetcrawl

meetcrawl has **no installable app yet**. The verification surface is the **repository itself**: required documentation, GitHub check wiring, and the same docs/secret gate CI runs until `go.mod` exists.

## Product definition of done (Phase 1+, not runnable today)

When `go.mod` and `scripts/proof.sh` exist, a product change is done only if this exits **0** on a clean Linux runner with **no network credentials**:

```bash
make check && scripts/proof.sh
```

That run writes `proof/summary.json` and `proof/*.log` (gitignored). It must satisfy all **seven** headless checks listed in [docs/SPEC.md](../../docs/SPEC.md) section **9. Done = PR + headless proof**.

Until those files exist, **Doctor reports Phase 1 proof as not runnable** and Drive must not claim section 9 checks passed.

## Launch

Set an isolated run context (no servers, no credentials):

```bash
export REPO_ROOT="$(git rev-parse --show-toplevel)"
export RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)-$$}"
export EVIDENCE_DIR="${EVIDENCE_DIR:-/tmp/meetcrawl-verify-evidence/$RUN_ID}"
mkdir -p "$EVIDENCE_DIR"
cd "$REPO_ROOT"
```

Teardown is only env/state you created in this run (see Cleanup). There is nothing to keep alive between Drive steps.

## Doctor

Read-only preflight before Drive:

```bash
"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" doctor
```

Doctor must report:

- Current git root matches `REPO_ROOT`.
- `docs/SPEC.md` SHA-256 at run start (stored under `$EVIDENCE_DIR/spec.sha256`).
- **Phase 1 proof:** `not runnable` when `go.mod` or `scripts/proof.sh` is missing; `runnable` when both exist (then Drive should use the Phase 1 feature file, not today's doc checks alone).

If Doctor exits non-zero, fix the repo or paths before Drive.

## Drive

Read the feature map index first:

`.cursor/skills/verify-meetcrawl/features/README.md`

Pick **one** feature file and follow its **Driving** section. For today's repo, default to [required-docs-and-ci](./features/required-docs-and-ci.md).

When `go.mod` and `scripts/proof.sh` both exist, switch default Drive to [phase1-headless-proof](./features/phase1-headless-proof.md) and run:

```bash
make check && scripts/proof.sh
```

instead of doc-only checks. Until then, use the helper for mapped repo features:

```bash
"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" drive --feature <feature-id>
```

Feature IDs match the map filenames without `.md` (e.g. `required-docs-and-ci`, `secrets-and-tracked-forbidden`, `spec-integrity`, `phase1-headless-proof`).

## Evidence

Capture proof outside gitignored `proof/` so cleanup does not erase it.

- Directory: **`$EVIDENCE_DIR`** (default under `/tmp/meetcrawl-verify-evidence/<RUN_ID>/`).
- Always keep: `doctor.log`, `drive.log`, `spec.sha256`, and any feature-specific outputs named in the feature file.
- Append a one-line summary to `$EVIDENCE_DIR/summary.txt` with feature ID, exit code, and timestamp.

Standards: run the real commands from the feature file; do not mark a check passed without executing it. For Phase 1, exercise the user-facing proof script path, not internal test-only shortcuts.

## Cleanup

Remove only ephemeral run state **you** created. **Do not** delete `$EVIDENCE_DIR`.

```bash
"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" cleanup
```

Cleanup clears a temp workdir under `/tmp/meetcrawl-verify-scratch/$RUN_ID` if present. It does not stop processes (none are started) and does not remove evidence.

After cleanup, confirm evidence still exists:

```bash
test -f "$EVIDENCE_DIR/summary.txt" && test -f "$EVIDENCE_DIR/doctor.log"
```

## Helpers

Executable driver (repo checks and logging):

```bash
"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" doctor
"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" drive --feature required-docs-and-ci
"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" cleanup
```

Subcommands: `doctor`, `drive --feature <id>`, `cleanup`, `docs-secret-check` (same gate as CI when `go.mod` is absent).

Maintenance: when product behavior ships, update the feature map and default Drive feature; keep section 9 wording aligned with [docs/SPEC.md](../../docs/SPEC.md).
