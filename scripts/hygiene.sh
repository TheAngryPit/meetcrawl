#!/usr/bin/env bash
# Repository hygiene: required docs, no forbidden tracked files, no secret material.
# Only synthetic SQLite fixtures under testdata/ may be tracked.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

fail=0

for f in README.md docs/SPEC.md SECURITY.md CONTRIBUTING.md LICENSE; do
  if [ ! -f "$f" ]; then
    echo "missing $f"
    fail=1
  fi
done

while IFS= read -r -d '' path; do
  case "$path" in
    testdata/*.db|testdata/*.db-wal|testdata/*.db-shm) continue ;;
  esac
  base="${path##*/}"
  case "$base" in
    .env*|*.db|*.db-wal|*.db-shm|*credential*.json|client_secret*.json|*oauth*token*.json|token.json|tokens.json)
      echo "tracked forbidden file: $path"
      fail=1
      ;;
  esac
done < <(git ls-files -z)

# git grep exits 1 when nothing matches. That is success.
grep_status=0
git grep -I -n -E -e 'AKIA[0-9A-Z]{16}|-----BEGIN (RSA |OPENSSH |EC |DSA )?PRIVATE KEY-----|ghp_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|xox[baprs]-[0-9A-Za-z-]{10,}' -- . || grep_status=$?
if [ "$grep_status" -eq 0 ]; then
  echo "possible secret material in tracked files"
  fail=1
elif [ "$grep_status" -ne 1 ]; then
  echo "git grep failed with status $grep_status"
  exit "$grep_status"
fi

if [ "$fail" -ne 0 ]; then
  exit "$fail"
fi
echo "hygiene check passed"
