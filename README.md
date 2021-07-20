# oneshow 一个简单的onedrive第三方命令行工具

# 功能

1.支持多帐号，可用su命令进行切换  
2.支持通过ls命令，浏览网盘内的文件  
3.支持文件的批量下载和上传，支持断点继传或下载  
4.支持对文件搜索  
5.支持单个移动文件或文件夹  
6.删除文件到回收站  
7.支持linux和windows系统  

# 使用
直接执行oneshow,查看支持的命令，支持使用-h查看子命令的详细使用方法。  
执行oneshow回车
```
HELP ===========================
oneshow version v0.1.1 build time:2021-07-20 12:56:53 
================================


info            show onedrive info

d               download a file or dir or URL to local

auth            get a auth for new user

u               upload a file or dir to onedrive

web             run this http super serivce (beta version)

su              swich to other logined user

saveUser        save current user to name

rm              remove a file or dir to trash

users           list login users

search          search files by key

mv              move file to other dir

ls              list onedrive path

```

首先通过oneshow auth增加一个用户的帐号配置，通过onedrive的授权后，配置文件保存在用户目录的~/.od.json文件中,然后通过oneshow saveUser alias，就保存了一个别名为alias的配置文件，当要使用这个账户时候通过oneshow su alias,切换到这个用户。

其他命令的用法，比如查询ls子命令帮助，可通过执行 oneshow ls -h 查看命令的详细使用方法。

```
usage: ls [OPTION] path

list onedrive path

-h  print help

-l  list files detail
```
其中path参数一定要为onedrive全路径,例如查询根目录，执行oneshow ls -l /  

# 构建

建议在linux下进行构建代码，其他平台没有进行过测试。下载代码后直接进入main文件夹执行make即可。
