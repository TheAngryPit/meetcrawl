#!/usr/bin/env bash
# Phase 1 headless proof (docs/SPEC.md section 9). Synthetic fixtures only.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROOF_DIR="${REPO_ROOT}/proof"
BUILD_HOME="${HOME}"
PROOF_HOME="$(mktemp -d)"
export HOME="${PROOF_HOME}"
export GOMODCACHE="${BUILD_HOME}/go/pkg/mod"
export GOPATH="${BUILD_HOME}/go"

MEETCRAWL_CFG="${HOME}/.config/meetcrawl/config.toml"
INDEX_DB="${HOME}/.local/share/meetcrawl/meetcrawl.db"
READS_DB="${HOME}/.local/share/meetcrawl/reads.db"

WHISP_SUPPORTED_DB="${REPO_ROOT}/testdata/fixtures/openwhispr/supported/transcriptions.db"
WHISP_UNSUPPORTED_DB="${REPO_ROOT}/testdata/fixtures/openwhispr/unsupported/transcriptions.db"
GMEET_SUPPORTED_FIX="${REPO_ROOT}/testdata/fixtures/gdrive/supported"
GMEET_UNSUPPORTED_FIX="${REPO_ROOT}/testdata/fixtures/gdrive/unsupported"

mkdir -p "${PROOF_DIR}"
BINDIR="${PROOF_HOME}/bin"
mkdir -p "${BINDIR}"

log() {
  echo "[proof] $*" | tee -a "${PROOF_DIR}/proof.log"
}

fail() {
  log "FAIL: $*"
  exit 1
}

json_get() {
  local path="$1"
  local json="$2"
  PATH_VAL="${path}" JSON_DOC="${json}" python3 <<'PY'
import json, os
doc = json.loads(os.environ["JSON_DOC"])
cur = doc
for key in os.environ["PATH_VAL"].split("."):
    cur = cur[key]
print(cur)
PY
}

fixture_sha256_file="${PROOF_DIR}/fixtures.sha256"
record_fixture_hashes() {
  find "${REPO_ROOT}/testdata/fixtures" -type f ! -name '.gitkeep' | sort | xargs sha256sum >"$1"
}

assert_fixture_hashes_unchanged() {
  local before="$1" after="$2"
  if ! diff -q "$before" "$after" >/dev/null; then
    diff -u "$before" "$after" | tee -a "${PROOF_DIR}/fixtures-diff.log" || true
    fail "fixture sha256 changed after sync/index"
  fi
  log "fixture sha256 unchanged"
}

build_binaries() {
  log "building binaries"
  (
    cd "${REPO_ROOT}"
    HOME="${BUILD_HOME}" GOWORK=off go build -o "${BINDIR}/whispcrawl" ./cmd/whispcrawl
    HOME="${BUILD_HOME}" GOWORK=off go build -o "${BINDIR}/gmeetcrawl" ./cmd/gmeetcrawl
    HOME="${BUILD_HOME}" GOWORK=off go build -o "${BINDIR}/meetcrawl" ./cmd/meetcrawl
  ) >>"${PROOF_DIR}/build.log" 2>&1
}

run_sync_ingest() {
  log "check 1: crawler sync ingest counts"
  "${BINDIR}/whispcrawl" init >>"${PROOF_DIR}/whisp-init.log" 2>&1
  local whisp_out
  whisp_out="$("${BINDIR}/whispcrawl" --json sync --source-db "${WHISP_SUPPORTED_DB}" 2>>"${PROOF_DIR}/whisp-sync.log")"
  echo "${whisp_out}" >>"${PROOF_DIR}/whisp-sync.log"
  local whisp_artifacts
  whisp_artifacts="$(json_get result.artifacts "${whisp_out}")"
  [[ "${whisp_artifacts}" == "3" ]] || fail "whispcrawl artifacts=${whisp_artifacts} want 3"

  "${BINDIR}/gmeetcrawl" init >>"${PROOF_DIR}/gmeet-init.log" 2>&1
  local gmeet_out
  gmeet_out="$("${BINDIR}/gmeetcrawl" --json sync --fixture "${GMEET_SUPPORTED_FIX}" 2>>"${PROOF_DIR}/gmeet-sync.log")"
  echo "${gmeet_out}" >>"${PROOF_DIR}/gmeet-sync.log"
  local gmeet_artifacts
  gmeet_artifacts="$(json_get result.artifacts "${gmeet_out}")"
  [[ "${gmeet_artifacts}" == "3" ]] || fail "gmeetcrawl artifacts=${gmeet_artifacts} want 3"
  log "sync ingest ok (3 + 3 artifacts)"
}

patch_meetcrawl_privacy() {
  [[ -f "${MEETCRAWL_CFG}" ]] || fail "meetcrawl config missing at ${MEETCRAWL_CFG}"
  python3 <<PY
from pathlib import Path
p = Path("${MEETCRAWL_CFG}")
text = p.read_text()
block = '''[[privacy.rules]]
calendar_ical = "cal-synthetic-001"
class = "restricted"
'''
if "calendar_ical = \"cal-synthetic-001\"" not in text:
    text = text.replace("rules = []", block.strip())
    p.write_text(text)
PY
}

run_index_checks() {
  log "check 2-3: index dedup, adhoc, pt-PT search"
  "${BINDIR}/meetcrawl" init >>"${PROOF_DIR}/meetcrawl-init.log" 2>&1
  patch_meetcrawl_privacy
  local idx_out
  idx_out="$("${BINDIR}/meetcrawl" --json index 2>>"${PROOF_DIR}/index.log")"
  echo "${idx_out}" >>"${PROOF_DIR}/index.log"
  local meetings
  meetings="$(json_get result.meetings "${idx_out}")"
  [[ "${meetings}" == "2" ]] || fail "index meetings=${meetings} want 2"
  [[ -f "${INDEX_DB}" ]] || fail "index db missing at ${INDEX_DB}"

  local dedup_fidelity adhoc_count
  dedup_fidelity="$(sqlite3 "${INDEX_DB}" "select best_fidelity from meetings where meeting_id not like 'adhoc:%' limit 1;")"
  [[ "${dedup_fidelity}" == "transcript" ]] || fail "dedup best_fidelity=${dedup_fidelity} want transcript"
  adhoc_count="$(sqlite3 "${INDEX_DB}" "select count(*) from meetings where meeting_id like 'adhoc:%';")"
  [[ "${adhoc_count}" == "1" ]] || fail "adhoc meetings=${adhoc_count} want 1"

  local search_out
  search_out="$("${BINDIR}/meetcrawl" --json search reuniao 2>>"${PROOF_DIR}/search.log")"
  echo "${search_out}" >>"${PROOF_DIR}/search.log"
  SEARCH_JSON="${search_out}" python3 <<'PY' || fail "pt-PT search missed accented fixture text"
import json, os
doc = json.loads(os.environ["SEARCH_JSON"])
hits = doc.get("result", {}).get("hits") or doc.get("hits")
if not hits:
    raise SystemExit(1)
snip = (hits[0].get("snippet") or "").lower()
if "reuni" not in snip:
    raise SystemExit(2)
PY
  log "index dedup and pt-PT search ok"
}

run_mcp_check() {
  log "check 4: MCP stdio session"
  export PROOF_MCP_CFG="${MEETCRAWL_CFG}"
  export PROOF_MCP_READS="${READS_DB}"
  export PROOF_MCP_BIN="${BINDIR}/meetcrawl"
  (
    cd "${REPO_ROOT}"
    HOME="${BUILD_HOME}" GOWORK=off go run ./scripts/proof_mcp.go
  ) >>"${PROOF_DIR}/mcp.log" 2>&1
  log "MCP checks ok"
}

run_rebuild_hash() {
  log "check 5: rebuildable ordered-row-dump hash"
  local hash1 hash2
  hash1="$(
    cd "${REPO_ROOT}"
    HOME="${BUILD_HOME}" GOWORK=off go run ./scripts/proof_index_hash.go "${INDEX_DB}"
  )"
  rm -f "${INDEX_DB}"
  "${BINDIR}/meetcrawl" --json index >>"${PROOF_DIR}/reindex.log" 2>&1
  hash2="$(
    cd "${REPO_ROOT}"
    HOME="${BUILD_HOME}" GOWORK=off go run ./scripts/proof_index_hash.go "${INDEX_DB}"
  )"
  [[ "${hash1}" == "${hash2}" ]] || fail "reindex hash mismatch ${hash1} vs ${hash2}"
  log "rebuild hash stable: ${hash1}"
  echo "${hash1}" >"${PROOF_DIR}/index-dump.hash"
}

run_metadata_deps_check() {
  log "check 6: metadata control.v1 and no crawlkit/remote deps"
  local meta_out schema
  meta_out="$("${BINDIR}/meetcrawl" --json metadata 2>>"${PROOF_DIR}/metadata.log")"
  echo "${meta_out}" >>"${PROOF_DIR}/metadata.log"
  schema="$(json_get schema_version "${meta_out}")"
  [[ "${schema}" == "crawlkit.control.v1" ]] || fail "metadata schema_version=${schema} want crawlkit.control.v1"
  if (
    cd "${REPO_ROOT}"
    HOME="${BUILD_HOME}" GOWORK=off go list -deps ./...
  ) | grep -Fq 'github.com/openclaw/crawlkit/remote'; then
    fail "go list -deps contains crawlkit/remote"
  fi
  log "metadata control.v1 and deps ok (no crawlkit/remote)"
}

run_unsupported_schema() {
  log "check 7: unsupported_schema fail-closed"
  local iso_home whisp_db gmeet_db

  iso_home="$(mktemp -d)"
  HOME="${iso_home}" "${BINDIR}/whispcrawl" init >>"${PROOF_DIR}/whisp-unsupported-init.log" 2>&1
  whisp_db="${iso_home}/.local/share/whispcrawl/whispcrawl.db"
  local whisp_code=0 whisp_out
  set +e
  whisp_out="$(HOME="${iso_home}" "${BINDIR}/whispcrawl" --json sync --source-db "${WHISP_UNSUPPORTED_DB}" 2>&1)"
  whisp_code=$?
  set -e
  echo "${whisp_out}" >>"${PROOF_DIR}/whisp-unsupported.log"
  [[ "${whisp_code}" -ne 0 ]] || fail "whispcrawl unsupported fixture exited 0"
  printf '%s' "${whisp_out}" | grep -q unsupported_schema || fail "whispcrawl missing unsupported_schema"
  [[ ! -f "${whisp_db}" ]] || fail "whispcrawl wrote archive on unsupported schema"

  iso_home="$(mktemp -d)"
  HOME="${iso_home}" "${BINDIR}/gmeetcrawl" init >>"${PROOF_DIR}/gmeet-unsupported-init.log" 2>&1
  gmeet_db="${iso_home}/.local/share/gmeetcrawl/gmeetcrawl.db"
  local gmeet_code=0 gmeet_out
  set +e
  gmeet_out="$(HOME="${iso_home}" "${BINDIR}/gmeetcrawl" --json sync --fixture "${GMEET_UNSUPPORTED_FIX}" 2>&1)"
  gmeet_code=$?
  set -e
  echo "${gmeet_out}" >>"${PROOF_DIR}/gmeet-unsupported.log"
  [[ "${gmeet_code}" -ne 0 ]] || fail "gmeetcrawl unsupported fixture exited 0"
  printf '%s' "${gmeet_out}" | grep -q unsupported_schema || fail "gmeetcrawl missing unsupported_schema"
  [[ ! -f "${gmeet_db}" ]] || fail "gmeetcrawl wrote archive on unsupported schema"
  log "unsupported_schema fail-closed ok"
}

write_summary() {
  python3 - <<PY
import json, pathlib
root = pathlib.Path("${PROOF_DIR}")
summary = {
    "ok": True,
    "home": "${PROOF_HOME}",
    "checks": [
        "sync_ingest",
        "index_dedup_adhoc",
        "pt_pt_search",
        "mcp_stdio_read_log",
        "fixture_sha256_unchanged",
        "index_rebuild_hash",
        "metadata_control_v1_no_remote_deps",
        "unsupported_schema_fail_closed",
    ],
    "index_dump_hash": (root / "index-dump.hash").read_text().strip() if (root / "index-dump.hash").exists() else "",
    "metadata_check": "passed",
}
(root / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
PY
  log "wrote ${PROOF_DIR}/summary.json"
}

cleanup() {
  rm -rf "${PROOF_HOME}"
}
trap cleanup EXIT

main() {
  log "proof start repo=${REPO_ROOT} home=${PROOF_HOME}"
  record_fixture_hashes "${fixture_sha256_file}.before"
  build_binaries
  run_sync_ingest
  run_index_checks
  record_fixture_hashes "${fixture_sha256_file}.after"
  assert_fixture_hashes_unchanged "${fixture_sha256_file}.before" "${fixture_sha256_file}.after"
  run_mcp_check
  run_rebuild_hash
  run_metadata_deps_check
  run_unsupported_schema
  write_summary
  log "proof ok"
}

main "$@"
