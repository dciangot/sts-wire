all: build build-windows build-macos

#bindata:
#	go get -u github.com/go-bindata/go-bindata/...
#	go-bindata -o rclone_bin.go data/

#data, err := Asset("data/rclone")
#if err != nil {
#	// Asset was not found.
#}

build:
	go build -o sts-wire_linux

build-windows:
	env GOOS=windows CGO_ENABLED=0 go build -mod vendor -o sts-wire_windows.exe -v

build-macos:
	env GOOS=darwin CGO_ENABLED=0 go build -mod vendor -o sts-wire_osx -v