# third_party

`uploader` 以 Go module 形式嵌入，供 `-upload` 使用。

- `go.mod`：`replace uploader => ./third_party/uploader`
- 同步上游（本机路径按实际调整）：

```shell
rsync -a --delete \
  --exclude '.git' --exclude 'rust' --exclude 'bin' --exclude 'docs' \
  --exclude 'cmd' --exclude 'scripts' --exclude 'main.go' \
  /Users/simon/Documents/tools/uploader/ third_party/uploader/
```
