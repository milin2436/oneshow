# =============================================================================
# oneshow 构建
#
# 分支历史混乱，这是合并后的正确版本。修掉的三处历史错误：
#
#   1) ver2 分支（a8a2f42）用 -X '.../oneshow/v2/cmd.VERSION=...' 是错的：
#      go.mod 是 module github.com/milin2436/oneshow，没有 /v2 后缀，
#      那个符号不存在，-X 静默失败，版本号永远停在源码默认值 "unknown"。
#      实测对照：
#         正确路径 .../oneshow/cmd.VERSION  -> 打印 "NAME-A VER-A"
#         错误路径 .../oneshow/v2/cmd.VERSION -> 打印 "NAME-B unknown"
#
#   2) f6fa611 的 android 目标只有 GOOS/GOARCH，没有 CGO_ENABLED=1。
#      交叉编译时 Go 默认把 CGO 关掉，DNS 退回纯 Go 解析器读不存在的
#      /etc/resolv.conf，在手机上必然报 connection refused。
#
#   3) 手机上的 HOME 是空的，程序去找 /.config/oneshow 直接失败，
#      android-run 里显式 export HOME。
#
# 用法：
#   make                    主机 linux/amd64
#   make android            Android arm64（真机）
#   make android-push       编 + 推到手机
#   make android-run ARGS="users"   推到手机直接跑
#   make pkg                打发行包（linux + windows zip）
# =============================================================================

# 版本号从 git 来，保证显示的和编译的代码一致：
#   tag 上            -> v1.1.0
#   tag 后 N 个提交   -> v1.1.0-N-g<hash>
#   工作区有改动      -> 末尾加 -dirty
ver       := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
buildTime := $(shell date +"%F %T")

out  = oneshow
# go build 编译的是当前目录的全部 .go（不是只有 objs），
# 所以依赖必须用 wildcard，手写的文件清单漏一个就会编出旧二进制
objs = $(filter-out %_test.go,$(wildcard *.go))

# 版本注入，各目标共用。路径里绝对不能有 /v2 —— 见上面第 1 条。
VFLAG = -X 'github.com/milin2436/oneshow/cmd.VERSION=$(ver) build time:$(buildTime)' -X 'github.com/milin2436/oneshow/cmd.CmdName=oneshow'

# ---------- Android NDK 交叉编译 ----------
# Go 1.26 已经移除了 NDK 自动探测（src 里搜不到 ANDROID_NDK / ANDROID_SDK），
# 只设 ANDROID_NDK_ROOT 会退回主机的 gcc，然后报 android/log.h 找不到。
# 所以 CC / CGO_CFLAGS / CGO_LDFLAGS 必须显式给，下面三点少一个都不行：
#   1) CGO_ENABLED=1        交叉编译时 Go 默认 CGO=0，那是 DNS 挂的根因
#   2) 用 aarch64- 的 wrapper  x86_64- 那个编出来 arm64 真机跑不了
#   3) 不要加 -D__ANDROID_API__  wrapper 的 --target 已经定义了，重复会 -Werror
#
# -Wl,--as-needed 把 C++ 运行时整个裁掉，产物只依赖 /system/lib64 的 bionic，
# 不需要额外带 libc++_shared.so，单文件部署。
NDK   = /home/super/Android/Sdk/ndk/29.0.13846066
PRE   = $(NDK)/toolchains/llvm/prebuilt/linux-x86_64
API   = 29

ANDROID_CC      = $(PRE)/bin/aarch64-linux-android$(API)-clang
ANDROID_CXX     = $(PRE)/bin/aarch64-linux-android$(API)-clang++
ANDROID_CFLAGS  = --sysroot=$(PRE)/sysroot -target aarch64-linux-android$(API) -O2 -g -fdata-sections -ffunction-sections
ANDROID_LDFLAGS = --sysroot=$(PRE)/sysroot -target aarch64-linux-android$(API) -O2 -g -L$(PRE)/sysroot/usr/lib/aarch64-linux-android/$(API) -Wl,--build-id=none -Wl,--exclude-libs,ALL -Wl,--gc-sections -Wl,--as-needed -Wl,-z,noexecstack


# ---------- 手机 ----------
# 手机上 HOME 是空的，所以给个可写目录，否则程序启动就去找 /.config/oneshow
PHONE_HOME = /data/local/tmp/home
PHONE_DIR  = /data/local/tmp/oneshow
PHONE_BIN  = $(PHONE_DIR)/$(out)-android
ADB        = adb$(if $(DEVICE), -s $(DEVICE))

.PHONY : all clean install pkg win arm64 static android android-push android-run

all : $(out)

# 默认：主机 linux/amd64（带 cgo）
$(out) : $(objs)
	echo "ver=$(ver)  $(buildTime)"
	go build -ldflags "-s -w $(VFLAG)" -o $(out)

clean :
	rm -f $(out) $(out).exe $(out)-arm $(out)-android $(out)-linux-amd64.zip $(out)-windows-386.zip

install :
	upx $(out)

pkg : $(out) win
	zip -r -0 $(out)-linux-amd64.zip $(out)
	zip -r -0 $(out)-windows-386.zip $(out).exe

# Windows 32 位
win : $(objs)
	GOOS=windows GOARCH=386 go build -ldflags "-s -w $(VFLAG)" -o $(out).exe

# ARM 服务器 / 树莓派
arm64 : $(objs)
	GOOS=linux GOARCH=arm64 go build -ldflags "-s -w $(VFLAG)" -o $(out)-arm

# 全静态无 cgo。注意两点：
#   * 输出文件名还是 $(out)，所以 "make static && make pkg" 能打出静态发行包
#   * 静态版手机上 DNS 会退回纯 Go 解析器，在 Android 上不可用
static : $(objs)
	CGO_ENABLED=0 go build -ldflags "-s -w $(VFLAG)" -o $(out)

# Android arm64：真机（OnePlus 9R 等），带 cgo，DNS 走 bionic getaddrinfo
android : $(objs)
	CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC="$(ANDROID_CC)" CXX="$(ANDROID_CXX)" CGO_CFLAGS="$(ANDROID_CFLAGS)" CGO_LDFLAGS="$(ANDROID_LDFLAGS)" go build -ldflags "-s -w $(VFLAG)" -o $(out)-android

# 推到手机
android-push : android
	$(ADB) shell mkdir -p $(PHONE_DIR) $(PHONE_HOME)
	$(ADB) push $(out)-android $(PHONE_BIN)
	$(ADB) shell chmod 755 $(PHONE_BIN)

# 推上去直接跑，ARGS 原样传给程序：make android-run ARGS="users"
android-run : android-push
	$(ADB) shell 'export HOME=$(PHONE_HOME); $(PHONE_BIN) $(ARGS)'
