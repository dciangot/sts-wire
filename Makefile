all: bindata build

bindata:
	${HOME}/go/bin/go-bindata data/

build:
	go build .
	