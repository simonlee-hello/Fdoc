# Fdoc
信息收集工具 对目标机器上的文档进行收集并打包

```
Usage of Fdoc:
  -d string
        root path to query (global option) (default "c:\\")
  -f string
        query files by filename (only for QueryByFileName),eg. '-f config  -f config,password,secret'
  -k string
        query files in content by keyword (only for QueryByKeyword),eg. '-k config -k password:,secret:,token:'
  -max string
        max file size can be zip (global option) (default "1GB")
  -o string
        zip output path (global option) (default "output.zip")
  -t string
        only query and pack files after the date,like '2023-10-01' (global option)(default "")
  -x string
        paths to skip query (global option) (default "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public")
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

