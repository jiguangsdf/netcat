CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a -ldflags '-s -w' -gcflags="all=-trimpath=${PWD}" -asmflags="all=-trimpath=${PWD}" -o bin/netcat_linux_amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -v -a -ldflags '-s -w' -gcflags="all=-trimpath=${PWD}" -asmflags="all=-trimpath=${PWD}" -o bin/netcat_darwin_amd64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -v -a -ldflags '-s -w' -gcflags="all=-trimpath=${PWD}" -asmflags="all=-trimpath=${PWD}" -o bin/netcat_linux_arm64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -v -a -ldflags "-s -w" -gcflags="all=-trimpath=${PWD}" -asmflags="all=-trimpath=${PWD}" -o bin/netcat_windows_amd64.exe
