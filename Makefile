.PHONY: build test lint install clean

build:
	go build -o bin/find-me-gluten-free-pp-cli ./cmd/find-me-gluten-free-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/find-me-gluten-free-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/find-me-gluten-free-pp-mcp ./cmd/find-me-gluten-free-pp-mcp

install-mcp:
	go install ./cmd/find-me-gluten-free-pp-mcp

build-all: build build-mcp
