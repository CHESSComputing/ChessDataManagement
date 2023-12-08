flags=-ldflags="-s -w"
# flags=-ldflags="-s -w -extldflags -static"
TAG := $(shell git tag | sed -e "s,v,,g" | sort -r | head -n 1)

all: build

build:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" chess_client.go
endif
	go clean; rm -rf pkg; go build -o chess_client ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" chess_client.go
endif

build_all: build_darwin_amd64 build_darwin_arm64 build_amd64 build_arm64 build_power8 build_windows

build_darwin_amd64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" chess_client.go
endif
	go clean; rm -rf pkg chess_client_darwin; GOOS=darwin go build -o chess_client ${flags}
	mv chess_client chess_client_darwin_amd64
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" chess_client.go
endif

build_darwin_arm64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" chess_client.go
endif
	go clean; rm -rf pkg chess_client_darwin; GOARCH=arm64 GOOS=darwin go build -o chess_client ${flags}
	mv chess_client chess_client_darwin_arm64
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" chess_client.go
endif

build_amd64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" chess_client.go
endif
	go clean; rm -rf pkg chess_client_linux; GOOS=linux go build -o chess_client ${flags}
	mv chess_client chess_client_amd64
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" chess_client.go
endif

build_power8:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" chess_client.go
endif
	go clean; rm -rf pkg chess_client_power8; GOARCH=ppc64le GOOS=linux go build -o chess_client ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" chess_client.go
endif
	mv chess_client chess_client_power8

build_arm64:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" chess_client.go
endif
	go clean; rm -rf pkg chess_client_arm64; GOARCH=arm64 GOOS=linux go build -o chess_client ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" chess_client.go
endif
	mv chess_client chess_client_arm64

build_windows:
ifdef TAG
	sed -i -e "s,{{VERSION}},$(TAG),g" chess_client.go
endif
	go clean; rm -rf pkg chess_client.exe; GOARCH=amd64 GOOS=windows go build -o chess_client.exe ${flags}
ifdef TAG
	sed -i -e "s,$(TAG),{{VERSION}},g" chess_client.go
endif

install:
	go install

clean:
	go clean; rm -rf pkg
