buildTime=$(shell date +"%F %T")
out=oneshow
objs=oneshow.go web.go user.go
$(out) : $(objs)
	echo $(buildTime)
	go build  -ldflags "-s -w -X 'github.com/milin2436/oneshow/cmd.VERSION=v0.1.2 build time:$(buildTime)' -X 'github.com/milin2436/oneshow/cmd.CmdName=oneshow' "  -o $(out)
.PHONY : win
.PHONY : static
clean :
	rm -rf $(out)
	rm -rf $(out).exe
install :
	upx $(out)
win :
	GOOS=windows GOARCH=386 go build  -ldflags "-s -w -X 'github.com/milin2436/oneshow/cmd.VERSION=v0.1.2 build time:$(buildTime)' -X 'github.com/milin2436/oneshow/cmd.CmdName=oneshow' "  -o $(out).exe
static :
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(out)
.PHONY : pkg
pkg : $(out) win
	zip -r -0 $(out)-linux-amd64.zip $(out)
	zip -r -0 $(out)-windows-386.zip $(out).exe
