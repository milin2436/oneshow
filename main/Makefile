buildTime=$(shell date +"%F %T")
out=oneshow
objs=oneshow.go web.go user.go
$(out) : $(objs)
	echo $(buildTime)
	go build  -ldflags "-s -w -X 'github.com/milin2436/oneshow/cmd.VERSION=v0.1.1 build time:$(buildTime)' -X 'github.com/milin2436/oneshow/cmd.CmdName=oneshow' "  -o $(out)
.PHONY : win
.PHONY : static
clean :
	rm -rf $(out)
install :
	upx $(out)
win :
	GOOS=windows GOARCH=amd64 go build  -ldflags "-s -w -X 'github.com/milin2436/oneshow/cmd.VERSION=v0.1.1 build time:$(buildTime)' -X 'github.com/milin2436/oneshow/cmd.CmdName=oneshow' "  -o $(out).exe
static :
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(out)
