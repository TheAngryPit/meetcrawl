# Required docs and CI wiring

A contributor (or agent) can confirm the spec-stage repo ships the required documentation and GitHub automation files before any Go code lands.

## Sub-features

- `docs-core` — README, SPEC, SECURITY, CONTRIBUTING present.
- `github-governance` — CODEOWNERS and Dependabot config present.
- `check-workflow` — `.github/workflows/check.yml` present for the required `check` status.

## How to get to it (user POV)

- Clone or open the meetcrawl repository locally.
- Inspect the tree at the repository root and under `.github/`.

## Driving it with meetcrawl-verify

Preconditions:

- `meetcrawl-verify doctor` exited 0 and wrote `$EVIDENCE_DIR/spec.sha256`.

- **List required paths.** Run `"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" drive --feature required-docs-and-ci`. Stdout lists `present` for each required file or reports `missing`.
- **Proof.** Exit code 0 and `$EVIDENCE_DIR/drive.log` contains `present` for all seven paths.

## Gotchas

- LICENSE is enforced by CI but is not part of this feature ID; secrets feature runs the full docs list including LICENSE.
- When `go.mod` appears, CI also runs `make check`; this feature still only checks file presence.
