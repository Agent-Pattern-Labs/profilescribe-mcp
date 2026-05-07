.PHONY: build test fmt clean

build:
	mkdir -p bin
	go build -o bin/profilescribe-mcp ./cmd/profilescribe-mcp

test:
	go test ./...

fmt:
	gofmt -w cmd internal

clean:
	rm -rf bin dist

