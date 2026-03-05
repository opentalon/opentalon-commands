BINARY_NAME ?= opentalon-commands

build:
	go build -o $(BINARY_NAME) .

test:
	go test -race -count=1 -v ./...

lint:
	golangci-lint run

.PHONY: build test lint
