# Uploader

多后端文件上传 CLI。拷贝单个二进制到目标机，即可把文件或目录上传到临时网盘 / 文件托管，并拿到下载链接。

适合无头环境、脚本调用，以及授权渗透测试中「C2 不适合拉大文件」时的云端中转外带。

Inspired by / based on [Mikubill/transfer](https://github.com/Mikubill/transfer)（已基本停更），本仓库在其思路上做了渠道维护、无头部署与故障转移等改进。

## Features

- **多后端**：temp.sh、litterbox、gofile、文叔叔等；不传 `-b` 时先 probe 再按延迟择路，传 `-b` 则固定渠道
- **目录友好**：目录默认先 zip 再传；`-r` 可按文件逐个上传
- **无头友好**：`-q` 只把链接打到 stdout，方便脚本 / C2 回显
- **体积克制**：Go 版偏 stdlib，体积约数 MB 级，便于投递
- **可选加密**：上传前加密，下载后本地解密
- **探测**：`probe` 先测渠道可用性，再传真正重要的包

## Install / Build

```bash
# Go（默认）
make all
# 或
CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o uploader .

# 产物目录：bin/linux、bin/windows、bin/osx、bin/freebsd
```

可选 Rust 实现（实验）：

```bash
make build-rust
# 或交叉编译
make build-rust-cross
```

## Release

推送 `v*` 标签会触发 GitHub Actions，用 GoReleaser 自动构建多平台二进制并创建 Release（含 changelog）：

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Quick start

```bash
# 自动择路（默认：未传 -b → 先 probe，按延迟选最快且容量够的渠道）
uploader ./file.bin
uploader -q ./mydir

# 指定后端上传
uploader -b temp ./file.bin

# 目录：先压缩再上传
uploader -b lit ./mydir

# 仅探测
uploader backends
uploader probe
uploader probe temp lit gof -timeout 20
```

成功后 stdout（或默认输出）会给出下载链接；再用自己的环境从云端取回即可。

## Config

```bash
# 默认后端（未传 -b 时）
export UPLOADER_BACKEND=lit          # Linux / macOS
set UPLOADER_BACKEND=lit             # Windows cmd

# 配置文件
# Linux:   ~/.config/uploader/config
# Windows: %APPDATA%\uploader\config
# 示例:
#   backend=lit          # auto 模式下优先尝试的渠道

# 代理（对所有后端生效）
export https_proxy=http://127.0.0.1:6152
export http_proxy=http://127.0.0.1:6152
```

脚本里不要使用 `-keep`（会等待回车）。

## More commands

```bash
uploader -b lit -r ./mydir           # 目录内逐文件上传（不 zip）
uploader encrypt -k secret ./file
uploader decrypt -k secret -o out.bin ./file.encrypt
```

## Backend status

| Status | Meaning |
|--------|---------|
| ok | 稳定，默认池可用 |
| flaky | 不稳定；需要 `-force` |
| down | 已禁用；需要 `-force` |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | 成功 |
| 1 | 上传 / 配置错误 |
| 2 | 参数错误 |

## Library API（`uploader/route`）

除 CLI 外，可把本模块当库嵌入（例如 [Fdoc](https://github.com/simonlee-hello/Fdoc) 的 `-upload`）。

核心入口在 `uploader/route`：

| API | 说明 |
|-----|------|
| `UploadAuto(path, opts)` | 单文件上传；`opts.Backend` 为空则按体积 probe + failover |
| `UploadWithOptions(files, opts)` | 多路径；同上 |
| `FindBackend` / `Backends` / `ProbeRankedForUpload` | 渠道表与探测 |

```go
import "uploader/route"

link, backend, err := route.UploadAuto("/tmp/a.tar.gz", route.Options{
    // Backend: "",   // 空 = auto
    // Backend: "temp", // 钉死渠道（不 failover）
    Force: false,
    Quiet: true,
    Mute:  true, // 吞掉 PostUpload 的 stdout，只通过返回值拿链接
})
if err != nil {
    return err
}
_ = link    // https://...
_ = backend // 实际成功的短名，如 "temp"
```

`Options` 要点：

- `Backend` 为空 → auto（与 CLI 不传 `-b` 一致）
- `Mute`/`Quiet` → 适合库调用，避免污染调用方 stdout
- `Encrypt` + `Key` → 流式加密后再传
- `OnSuccess` → 成功回调（CLI 用来写 `last-backend`）

CLI 的 `cmd` 已改为调用同一套 `route` 逻辑，避免两套实现。文叔叔等后端在库路径下会补默认 `BlockSize`/`Parallel`，避免 Config 为零时 panic。

**并发说明**：
- `UploadAuto` / `UploadWithOptions` 同一进程内仍串行（`sessionMu`；真实上传会改全局 `TransferConfig`）。
- **auto / `probe` 探测已真正并发**（默认并行 4）：探测走 `UploadFileOpts` 隔离配置，不再改 `os.Stdout`；超时探测可立即返回，不必 join 慢后端。
- PostUpload 链接输出统一经 `apis.EmitLink`（Mute/Quiet 下静默），避免并发探测刷屏或抢 stdout。

依赖方式示例：

```bash
# go.mod
require uploader v0.0.0
replace uploader => ../uploader   # 或 git / vendor 路径
```

## Disclaimer

仅用于授权渗透测试、攻防演练与防御研究。请勿用于未授权系统。使用者自行承担合规与法律风险。

## Credits

- [Mikubill/transfer](https://github.com/Mikubill/transfer) — 原始多后端上传思路与实现基础

## License

[MIT](./LICENSE)
