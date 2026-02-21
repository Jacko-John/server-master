all: build

build: build-server build-client

build-server:
	CGO_ENABLED=0 go build -ldflags="-s -w -extldflags '-static'" -tags netgo -o ServerMaster ./cmd/server

build-client:
	CGO_ENABLED=0 go build -ldflags="-s -w -extldflags '-static'" -tags netgo -o SMClient ./cmd/client