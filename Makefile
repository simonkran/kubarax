BINARY_NAME=kubarax
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: build clean test lint run

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)
	go clean

test:
	go test ./... -v

lint:
	golangci-lint run ./...

run: build
	./$(BINARY_NAME)

install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy
