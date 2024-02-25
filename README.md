# oneshow 一个简单的onedrive第三方命令行工具

# 功能

1.支持多帐号，可用su命令进行当前用户切换  
2.支持通过ls命令，浏览网盘内的文件  
3.支持文件的批量下载和上传，支持断点继传或下载  
4.支持对文件搜索  
5.支持单个移动文件或文件夹  
6.删除文件到回收站  
7.以web方式浏览和下载网盘文件，实验阶段  
8.以webdav方式，挂载到设备，不支持修改文件和增加文件  
9.支持linux和windows系统，其他设备请自行构建   
10.支持通过配置文件，自定义使用自己的onedrive API   
11.支持webdav挂载时，通过配置加速地址，以达到加速下载onedrive文件的目的   
12.支持可断点下载的URL资源，直接上传到网盘，无需下载到本地后再上传  



# 使用
直接执行oneshow,查看支持的命令，支持使用-h查看子命令的详细使用方法。可自建 .oneshow.json 配置文件，配置加速下载和使用自己的onedrive API功能。    

执行oneshow回车
```
HELP ===========================
oneshow version v0.1.8 build time:2022-03-27 11:31:32 
================================


rm              remove a file or dir to trash

su              swich to other logined user

search          search files by key

ls              list onedrive path

users           list login users

u               upload a file or dir to onedrive

web             run this http super serivce (beta version)

webdav          run webdav service for onedirve (only read)(beta version)

info            show onedrive info

auth            get a auth for new user

saveUser        save current user to name

who             show current user name

mv              move file to other dir

d               download a file or dir or URL to local


```

首先通过oneshow auth增加一个用户的帐号配置，该操作请在桌面环境执行以及事先安装好浏览器，用户在浏览器上操作同意受权oneshow应用后，保存登录信息的文件会自动生成和保存在用户目录的~/.config/oneshow/.od.json文件中,其中~/.config/oneshow/用户的oneshow配置目录(所有配置皆放在这个目录下),然后通过oneshow saveUser alias，就保存了一个别名为alias的配置文件，当要使用这个账户时候通过oneshow su alias,切换到这个用户，主要通过配置目录下的.od_cur_user.id文件来指定当前用户，可通过oneshow who来查看当前用户是谁。

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

可通过环境变量oneshowuser来使用某个特定用户的onedrive，而忽略从.od_cur_user.id读取当前用户信息。比如存在u1用户，打印他的网盘配额信息，在linux下可使用命令:
```
oneshowuser=u1 ./oneshow info

```

删除目标文件/test，放入到回收站使用：
```
./oneshow rm /test
```
上传/test下所有文件到/other目录使用:
```
./oneshow u -s /test -f /other

```
下面命令是下载/test下的所有文件，到当前目录，可用-d ”下载目录“，来指定下载到目标目录:
```
./oneshow d /test

```

搜索网盘内关键字为key的所有名录和文件，加-d可显示文件或目录的全路径：
```
./oneshow search -d key

```
打开web服务，绑定到127.0.0.1:4444，通过访问，http://127.0.0.1:4444/vfs 浏览网盘和下载文件：
```
./oneshow web  -u 127.0.0.1:4444

```
开启一个webdav服务，绑定到127.0.0.1:4444，ubuntu下，打开文件管理的连接到服务器，通过dav://127.0.0.1:4444/ 就可将onedrive挂载到本地。该功能不支持上传功能和移动目录，支持设置密码认证，为了信息安全在远程启用webdav服务时候请开启https协议和加入帐号认证，浏览和下载、直接播放视频等功能经测试没有什么大问题。

```
./oneshow webdav -u 127.0.0.1:4444 -ss user_alias

```
下面命令为开启一个TLS通道webdav服务，认证的用户和密码为username、password，证书和密钥文件为test.crt、test.key，绑定的服务端口为4444，ss参数为指定的username,如果将多个用户挂载到同一个webdav进程以";"隔开多个用户，例如 -ss "u1;u2;u3"这样的形式，这样u1用户的webdav资源路径就为/u1

```
./oneshow webdav -user username -passwd password -c test.crt -k test.key -u :4444 -ss "u1;u2;u3"

```

由于在中国境内，没有onedrive服务器，访问国际版时速度非常不稳定，故提供一个加速接口功能。思路为将onedrive的下载链接，通过参数传递给加速的中间服务器，以达到加速目的。以我个人的经验，移动用户的最佳实践为cloudflare workers加上优选cloudflare CDN ip 这个方案非常稳定，在线看4k视频没有什么问题。oneshow程序的配置文件为.oneshow.json，请务必将该文件建在用户配置目录下。
配置文件为json格式，accelerated_api为加速地址，acceleration为加速开关，只有当acceleration为true而且accelerated_api不为空的情况下，才会生效加速功能。

```JSON
{
    "accelerated_api":"https://exsample.com/fetch?url=",
    "acceleration":false
}
```

可通过oneshow程序配置文件.oneshow.json，使用自己的onedrive API。比如使用rclone的API [配置文件](https://github.com/milin2436/oneshow/blob/main/conf/oneshow.json.rclone)

```JSON
{
    "accelerated_api":"https://exsample.com/fetch?url=",
    "acceleration":false,
    "client_id":"b15665d9-eda6-4092-8539-0eec376afd59",
    "client_secret":"qtyfaBBYA403=unZUP40~_#",
    "scope":"Files.Read Files.ReadWrite Files.Read.All Files.ReadWrite.All offline_access Sites.Read.All",
    "redirect_uri":"http://localhost:53682/"
}

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

