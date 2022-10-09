#GOPATH:=$(PWD):${GOPATH}
#export GOPATH
flags=-ldflags="-s -w"
# flags=-ldflags="-s -w -extldflags -static"
TAG := $(shell git tag | sed -e "s,v,,g" | sort -r | head -n 1)

all: build

build:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg; go build -o web ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_all: build_darwin_amd64 build_darwin_arm64 build_amd64 build_arm64 build_power8 build_windows

build_darwin_amd64:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg web_darwin; GOOS=darwin go build -o web ${flags}
	mv web web_darwin_amd64
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_darwin_arm64:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg web_darwin; GOARCH=arm64 GOOS=darwin go build -o web ${flags}
	mv web web_darwin_arm64
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_amd64:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg web_linux; GOOS=linux go build -o web ${flags}
	mv web web_amd64
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_power8:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg web_power8; GOARCH=ppc64le GOOS=linux go build -o web ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
	mv web web_power8

build_arm64:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg web_arm64; GOARCH=arm64 GOOS=linux go build -o web ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
	mv web web_arm64

build_windows:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg web.exe; GOARCH=amd64 GOOS=windows go build -o web.exe ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

install:
	go install

clean:
	go clean; rm -rf pkg

test : test1

test1:
	go test -test.v .
