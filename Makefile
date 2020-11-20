all: build build-windows build-macos

#bindata:
#	${HOME}/go/bin/go-bindata data/

build:
	go build -o sts-wire_linux

build-windows:
	env GOOS=windows CGO_ENABLED=0 go build -mod vendor -o sts-wire_windows.exe -v

build-macos:
	env GOOS=darwin CGO_ENABLED=0 go build -mod vendor -o sts-wire_osx -v