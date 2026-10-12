#!/usr/bin/env bash
set -euo pipefail

real_root="$(git rev-parse --show-toplevel)"
VERIFY="$real_root/.cursor/skills/verify-meetcrawl/bin/meetcrawl-verify"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

mkdir -p "$scratch/scripts"
cat >"$scratch/go.mod" <<'EOF'
module example.com/meetcrawl-verify-fake

go 1.22
EOF
printf '%s\n' '#!/usr/bin/env bash' 'exit 0' >"$scratch/scripts/proof.sh"
chmod +x "$scratch/scripts/proof.sh"
cat >"$scratch/Makefile" <<'EOF'
.PHONY: check
check:
	@echo "simulated make check failure" >&2
	@exit 42
EOF

export RUN_ID="drive-pipeline-test-$$"
export EVIDENCE_DIR="$scratch/evidence"
mkdir -p "$EVIDENCE_DIR"

REPO_ROOT="$real_root" "$VERIFY" doctor >/dev/null

export REPO_ROOT="$scratch"
set +e
out="$("$VERIFY" drive --feature phase1-headless-proof 2>&1)"
code=$?
set -e

if [[ "$code" -eq 0 ]]; then
  echo "expected non-zero exit when make check fails, got 0" >&2
  echo "$out" >&2
  exit 1
fi
if [[ "$out" == *"drive ok"* ]]; then
  echo "must not print drive ok when make check fails" >&2
  echo "$out" >&2
  exit 1
fi
echo "drive pipeline test passed (exit=$code)"
