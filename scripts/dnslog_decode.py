#!/usr/bin/env python3
"""Decode Fdoc DNS callback exfil queries.

Wire format (see pkg/callback):
  <seq>-<total>-<base32chunk>.<task_id>.<dns-base>
  payload = base32(task_id|host|url)  # no padding, lowercase

Usage:
  # paste DNSLog lines (hostnames or full table rows), then Ctrl-D
  python3 scripts/dnslog_decode.py

  # from file / stdin
  python3 scripts/dnslog_decode.py dnslog.txt
  pbpaste | python3 scripts/dnslog_decode.py

  # one-shot args
  python3 scripts/dnslog_decode.py \\
    0-4-xxxx.op42.xxx.dnslog.cn \\
    1-4-yyyy.op42.xxx.dnslog.cn
"""

from __future__ import annotations

import argparse
import base64
import re
import sys
from collections import defaultdict

# First label: seq-total-chunk (chunk is base32 alphabet)
LABEL_RE = re.compile(
    r"(?P<seq>\d+)-(?P<total>\d+)-(?P<chunk>[a-z2-7]+)\.(?P<task>[a-z0-9-]{1,63})\.",
    re.IGNORECASE,
)


def extract_parts(text: str) -> dict[str, dict[int, str]]:
    """task_id -> {seq: chunk}. Also records expected total per task."""
    by_task: dict[str, dict[int, str]] = defaultdict(dict)
    totals: dict[str, int] = {}

    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        # Prefer scanning the whole line so table columns still work.
        for m in LABEL_RE.finditer(line):
            seq = int(m.group("seq"))
            total = int(m.group("total"))
            chunk = m.group("chunk").lower()
            task = m.group("task").lower()
            by_task[task][seq] = chunk
            prev = totals.get(task)
            if prev is not None and prev != total:
                print(
                    f"warning: task {task!r} total mismatch {prev} vs {total}",
                    file=sys.stderr,
                )
            totals[task] = total

    # stash totals on a side channel via attribute
    extract_parts.totals = totals  # type: ignore[attr-defined]
    return by_task


def b32decode_nopad(s: str) -> bytes:
    s = s.upper()
    s += "=" * ((8 - len(s) % 8) % 8)
    return base64.b32decode(s)


def decode_task(task: str, parts: dict[int, str], total: int) -> str:
    missing = [i for i in range(total) if i not in parts]
    if missing:
        raise ValueError(f"task {task!r}: missing chunks {missing} (have {sorted(parts)})")
    enc = "".join(parts[i] for i in range(total))
    raw = b32decode_nopad(enc)
    return raw.decode("utf-8", errors="replace")


def format_payload(raw: str) -> str:
    # task_id|host|url  (url may contain '|')
    a, sep, rest = raw.partition("|")
    if not sep:
        return raw
    b, sep2, c = rest.partition("|")
    if not sep2:
        return f"task_id={a}\nraw={raw}"
    return f"task_id: {a}\nhost:    {b}\nurl:     {c}"


def main() -> int:
    ap = argparse.ArgumentParser(description="Decode Fdoc DNSLog callback queries")
    ap.add_argument(
        "inputs",
        nargs="*",
        help="DNSLog lines / hostnames, or a file path; omit to read stdin",
    )
    ap.add_argument("-q", "--quiet", action="store_true", help="print only url lines")
    args = ap.parse_args()

    chunks: list[str] = []
    if not args.inputs:
        chunks.append(sys.stdin.read())
    else:
        for item in args.inputs:
            try:
                with open(item, encoding="utf-8", errors="replace") as f:
                    chunks.append(f.read())
            except OSError:
                chunks.append(item)

    text = "\n".join(chunks)
    by_task = extract_parts(text)
    totals: dict[str, int] = getattr(extract_parts, "totals", {})

    if not by_task:
        print("no Fdoc DNS chunks found", file=sys.stderr)
        return 1

    rc = 0
    for task in sorted(by_task):
        total = totals.get(task)
        if total is None:
            # infer from max seq+1 if total label missing somehow
            total = max(by_task[task]) + 1
        try:
            raw = decode_task(task, by_task[task], total)
        except Exception as e:
            print(f"error: {e}", file=sys.stderr)
            rc = 1
            continue

        if args.quiet:
            parts = raw.split("|", 2)
            print(parts[2] if len(parts) == 3 else raw)
        else:
            print(f"=== task {task} ({len(by_task[task])}/{total} chunks) ===")
            print(format_payload(raw))
            print()
    return rc


if __name__ == "__main__":
    raise SystemExit(main())
