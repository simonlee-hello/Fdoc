# Manual test fixtures

Directory: `testdata/manual/src`

Defaults changed: `-e documents`, `-max-file 100MB`, OS skip lists, `-max` keeps partial archive (exit 2).

## Quick checks

```bash
cd testdata/manual
BIN=./Fdoc-macos

# default documents only
$BIN -d src -x skipme -o out/01-docs.tar.gz -q; echo exit:$?
tar -tzf out/01-docs.tar.gz

# measure respects max
$BIN -d src -e any -x skipme -max 100 -max-file 0 -size -q; echo exit:$?

# truncate keep
$BIN -d src -e txt -x skipme -max 40 -max-file 0 -o out/trunc.tar.gz -q; echo exit:$?
ls -l out/trunc.tar.gz
tar -tzf out/trunc.tar.gz
```

See `expected/` for older list-based cases (use matching `-e` explicitly).

## Automated smoke / regression

From repo root:

```bash
bash scripts/smoke_regress.sh           # offline (default)
ONLINE=1 bash scripts/smoke_regress.sh  # also hit real upload + local webhook
```
