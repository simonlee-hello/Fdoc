# Fdoc
信息收集工具 对目标机器上的文档进行收集并打包

```
Usage:
  -d string
        root path to query (short) (default "c:\\")
  -max string
        max file size can be zip (short) (default "1GB")
  -o string
        zip output path (short) (default "output.zip")
  -t string
        only query and pack files after the date,like '2023-10-01' (short)(default "")
  -x string
        paths to skip query (short) (default "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public")
```

默认进行文件后缀查询

```
".pdf", ".docx", ".doc", ".xlsx", ".xls", ".csv",".pptx", ".ppt", ".zip", ".rar", ".7z", ".tar", ".gz", ".tgz"
```

```
Fdoc -d C:\ -max 10GB -o output.zip
```

通过文件名进行近似查询

```
Fdoc -d C:\ -max 10GB -o output.zip -f password,secret,config
```

通过关键字进行查询（查询文件内容）

```
Fdoc -d C:\ -max 10GB -o output.zip -k password:,secret:,token:
```

