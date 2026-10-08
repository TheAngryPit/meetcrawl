# Secrets and forbidden tracked files

The repository must not track env files, SQLite archives, credential JSON, or obvious secret material in version control.

## Sub-features

- `required-docs-license` — README, SPEC, SECURITY, CONTRIBUTING, LICENSE on disk.
- `forbidden-paths` — no tracked `.env*`, `*.db*`, or OAuth/credential JSON patterns.
- `secret-grep` — git grep finds no AWS/GitHub/Slack key patterns in tracked content.

## How to get to it (user POV)

- Work in a local clone with no extra tracked secrets added for the run.

## Driving it with meetcrawl-verify

Preconditions:

- Doctor completed successfully.

- **Run CI-equivalent gate.** Run `"$REPO_ROOT/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify" drive --feature secrets-and-tracked-forbidden`. Mirrors the `Docs and secret check` step in `.github/workflows/check.yml` when `go.mod` is missing.
- **Proof.** Exit code 0, stdout ends with `docs and secret check passed`, log saved to `$EVIDENCE_DIR/drive.log`.

## Gotchas

- `git grep` exit 1 means no matches — that is success.
- Untracked local secrets do not fail this check; only **tracked** paths matter.
