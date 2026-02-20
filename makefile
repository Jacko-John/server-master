all:build

build:
	CGO_ENABLED=0 go build -ldflags="-s -w -extldflags '-static'" -tags netgo -o ServerMaster ./cmd/server