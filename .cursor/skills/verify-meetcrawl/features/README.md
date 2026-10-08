# meetcrawl verification map

This directory is the maintained source for verifying meetcrawl. There is no app to launch yet; recipes exercise the repo, CI gates, and (when Phase 1 lands) the headless proof in docs/SPEC.md section 9.

## Baseline preconditions

- Run from the repository root with a clean git view of tracked files.
- Export `RUN_ID` and `EVIDENCE_DIR` per [SKILL.md](../SKILL.md) Launch.
- Run `meetcrawl-verify doctor` and require a recorded `docs/SPEC.md` hash under `$EVIDENCE_DIR/spec.sha256`.
- Do not modify `docs/SPEC.md` during a verification run.

## Driving conventions

- Use `.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify` for doctor, drive, and cleanup.
- One Drive step maps to one feature ID unless the feature file says otherwise.
- Log output is tee'd to `$EVIDENCE_DIR/doctor.log` and `$EVIDENCE_DIR/drive.log`.

## Proof and skip reporting

- Repo checks must match what CI runs today (docs/secret gate in `.github/workflows/check.yml` when `go.mod` is absent).
- Phase 1 proof checks are **not drivable** until both `go.mod` and `scripts/proof.sh` exist; report `not runnable`, do not skip silently.
- Record feature ID and exit code in `$EVIDENCE_DIR/summary.txt`.

## Feature entry contract

Each feature file uses exactly four H2 sections: `Sub-features`, `How to get to it (user POV)`, `Driving it with meetcrawl-verify`, `Gotchas`.

## Features

- [required-docs-and-ci](./required-docs-and-ci.md) — README, spec, security, contributing, CODEOWNERS, dependabot, check workflow.
- [secrets-and-tracked-forbidden](./secrets-and-tracked-forbidden.md) — CI docs/secret check and forbidden tracked paths.
- [spec-integrity](./spec-integrity.md) — SPEC hash unchanged across the run.
- [phase1-headless-proof](./phase1-headless-proof.md) — section 9 proof script (not runnable until Phase 1 files land).
