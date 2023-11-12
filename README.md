# Fdoc
信息收集工具 对目标机器上的文档进行收集并打包

## 特色功能

- 打包目录里如有链接文件，会将链接的文件一起打包
- 可限定大小，当文件大于该值则不进行打包操作

## TODO

-[x] 增加只对指定目录压缩的功能 -d 就是指定目录；-f就是指定文件；-k就是文件内容；-e就是后缀
-[x] 增加模式选项：1、指定目录压缩；2、后缀压缩；3、近似文件名压缩；4、近似内容
-[ ] 文件多时，会栈溢出，需要边爬取文件边进行打包

```shell
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
  -size
        Calculate total size
  -t string
        only query and pack files after the date,like '2023-10-01' (global option)(default "")
  -x string
        paths to skip query (global option) (default "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public")
```

默认进行文件后缀查询

```
".pdf", ".docx", ".doc", ".xlsx", ".xls", ".csv",".pptx", ".ppt", ".zip", ".rar", ".7z", ".tar", ".gz", ".tgz"
```

```shell
Fdoc -d C:\test -max 10GB -o output.zip #打包C:\test文件夹下所有文件
Fdoc -d C:\test -max 10GB -o output.zip -e all #打包C:\test文件夹下所有符合以上后缀的文件
Fdoc -d C:\test -max 10GB -o output.zip -e pdf #打包C:\test文件夹下所有pdf后缀的文件
```

获取符合条件的文件的总大小

```shell
Fdoc -d C:\test --size
```


通过文件名进行近似查询

```shell
Fdoc -d C:\ -max 10GB -o output.zip -f password,secret,config
```

通过关键字进行查询（查询文件内容）

```shell
Fdoc -d C:\ -max 10GB -o output.zip -k password:,secret:,token:
```

