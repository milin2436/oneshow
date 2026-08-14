# Version is derived from git so the displayed version always matches the
# code being built: "v1.1.0" at a tag, "v1.1.0-N-g<hash>" N commits after a
# tag, "-dirty" when the working tree has uncommitted changes.
buildTime := $(shell date +"%F %T")
out = oneshow
objs = oneshow.go webUI.go
ver := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
ldflags := -s -w -X 'github.com/milin2436/oneshow/cmd.VERSION=$(ver) build time:$(buildTime)' -X 'github.com/milin2436/oneshow/cmd.CmdName=oneshow'

$(out) : $(objs)
	echo $(buildTime)
	go build  -ldflags "$(ldflags)"  -o $(out)
.PHONY : win
.PHONY : static
clean :
	rm -rf $(out)
	rm -rf $(out).exe
install :
	upx $(out)
win :
	GOOS=windows GOARCH=386 go build  -ldflags "$(ldflags)"  -o $(out).exe
arm64 :
	GOOS=linux GOARCH=arm64 go build  -ldflags "$(ldflags)"  -o $(out)-arm
android :
	GOOS=android GOARCH=arm64 go build  -ldflags "$(ldflags)"  -o $(out)-android
static :
	CGO_ENABLED=0 go build -ldflags "$(ldflags)" -o $(out)
.PHONY : pkg
pkg : $(out) win
	zip -r -0 $(out)-linux-amd64.zip $(out)
	zip -r -0 $(out)-windows-386.zip $(out).exe
