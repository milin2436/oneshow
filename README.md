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
oneshow version v0.1.7 build time:2022-03-17 13:32:18 
================================


ls              list onedrive path

rm              remove a file or dir to trash

auth            get a auth for new user

users           list login users

su              swich to other logined user

who             show current user name

info            show onedrive info

d               download a file or dir or URL to local

u               upload a file or dir to onedrive

web             run this http super serivce (beta version)

saveUser        save current user to name

search          search files by key

mv              move file to other dir


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

查看当前用户网盘容量和使用情况使用：

```
./oneshow info

```

查看当前使用的用户的别名:
```
./oneshow who

```

删除目标文件/test，放入到回收站使用：
```
./oneshow rm /test
```
上传/test下所有文件到/other目录使用:
```
./oneshow u -s /test -f /other

```
下面下载/test下的所有文件，到当前目录，可用-d ”下载目录“，来指定目录:
```
./oneshow d /test

```

搜索网盘内关键字为key的所有名录和文件，加-d可显示文件或目录的全路径：
```
./oneshow -d search key

```
打开web服务，绑定到127.0.0.1:4444，通过访问，http://127.0.0.1:4444/vfs 浏览网盘和下载文件：
```
./oneshow web  -u 127.0.0.1:4444

```

# 构建

建议在linux下进行构建代码，下载代码后直接进入main文件夹执行make即可。
发布的二进制程序提供了linux和windows版本，但是最近的更改都没有在windows上进行测试。
对于mac用户，请自己进行构建。

# 版权声明

该软件使用rclone部分代码，基于MIT协议做出该声明，这部分代码用于解析onedrive api返回的json数据对应的结构体。使用的代码为
https://github.com/rclone/rclone/blob/master/backend/onedrive/api/types.go
rclone项目地址：
https://github.com/rclone/rclone/

