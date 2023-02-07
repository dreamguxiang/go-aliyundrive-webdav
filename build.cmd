set CGO_ENABLED=0
set GOOS=linux
set GOARCH=arm
go build -ldflags="-s -w" -o go_aliyun_webdav-linux-arm

set CGO_ENABLED=0
set GOOS=windows
set GOARCH=arm
go build -ldflags="-s -w" -o go_aliyun_webdav-windows-arm.exe

set CGO_ENABLED=0
set GOOS=linux
set GOARCH=386
go build -ldflags="-s -w" -o go_aliyun_webdav-linux-386

set CGO_ENABLED=0
set GOOS=windows
set GOARCH=386
go build -ldflags="-s -w" -o go_aliyun_webdav-windows-386.exe

set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o go_aliyun_webdav-linux-amd64

set CGO_ENABLED=0
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w" -o go_aliyun_webdav-darwin-amd64

set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o go_aliyun_webdav-windows-amd64.exe