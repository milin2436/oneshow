# oneshow 一个简单的onedrive第三方命令行工具

# 特色

1.支持多帐号，可用su命令进行切换  
2.支持通过ls命令，浏览网盘内的文件  
3.支持文件的批量下载和上传  
4.支持对文件搜索  
5.支持单个移动文件或文件夹  

# 使用
直接执行oneshow,查看支持的命令，支持使用-h查看子命令的详细使用方法。  
执行oneshow回车
```
HELP ===========================
mon version 2021-05-18 22:04:16 
================================


u               upload a file or dir to onedrive

search          search files by key

mv              move file to other dir

ls              list onedrive path

d               download a file or dir or URL to local

users           list login users

update          update token

uu              download myself

auth            get a auth for new user

upload          upload a little text file to onedrive

su              swich to other logined user

info            show onedrive info

web             run this http super serivce

saveUser        save current user to name

rm              remove a file or dir to trash
```

通过oneshow auth增加一个用户的帐号配置
# 构建

建议在linux下进行构建代码，其他平台没有进行过测试。下载代码后直接进入main文件夹执行make即可。
