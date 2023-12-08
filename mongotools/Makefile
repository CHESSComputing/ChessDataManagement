flags=-ldflags="-s -w"
# flags=-ldflags="-s -w -extldflags -static"
TAG := $(shell git tag | sed -e "s,v,,g" | sort -r | head -n 1)

all: build

build:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
endif
	go clean; rm -rf pkg; go build -o mongotools ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
endif

build_all: build_darwin_amd64 build_darwin_arm64 build_amd64 build_arm64 build_power8 build_windows

build_darwin_amd64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
endif
	go clean; rm -rf pkg mongotools_darwin; GOOS=darwin go build -o mongotools ${flags}
	mv mongotools mongotools_darwin_amd64
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
endif

build_darwin_arm64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
endif
	go clean; rm -rf pkg mongotools_darwin; GOARCH=arm64 GOOS=darwin go build -o mongotools ${flags}
	mv mongotools mongotools_darwin_arm64
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
endif

build_amd64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
endif
	go clean; rm -rf pkg mongotools_linux; GOOS=linux go build -o mongotools ${flags}
	mv mongotools mongotools_amd64
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
endif

build_power8:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
endif
	go clean; rm -rf pkg mongotools_power8; GOARCH=ppc64le GOOS=linux go build -o mongotools ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
endif
	mv mongotools mongotools_power8

build_arm64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
endif
	go clean; rm -rf pkg mongotools_arm64; GOARCH=arm64 GOOS=linux go build -o mongotools ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
endif
	mv mongotools mongotools_arm64

build_windows:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
endif
	go clean; rm -rf pkg mongotools.exe; GOARCH=amd64 GOOS=windows go build -o mongotools.exe ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go
endif

install:
	go install

clean:
	go clean; rm -rf pkg
