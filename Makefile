.PHONY: build test lint fmt install

build:
	go build -o gqurl .

install:
	go install .

test:
	go test -shuffle on ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
