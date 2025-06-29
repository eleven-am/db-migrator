.PHONY: test test-unit test-integration test-all coverage clean

# Run unit tests only
test-unit:
	go test -v ./parser ./generator ./diff ./introspect

# Run integration tests (requires PostgreSQL)
test-integration:
	go test -v -tags=integration ./...

# Run all tests
test: test-unit

test-all: test-unit test-integration

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./parser ./generator ./diff ./introspect
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run specific package tests
test-parser:
	go test -v ./parser

test-generator:
	go test -v ./generator

test-diff:
	go test -v ./diff

test-introspect:
	go test -v ./introspect

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html
	rm -f tools/db-migrator/parser/struct_parser_test_new.go

# Run linter
lint:
	golangci-lint run ./...

# Build the binary
build:
	go build -o bin/db-migrator .

# Install the binary
install: build
	cp bin/db-migrator $(GOPATH)/bin/

.DEFAULT_GOAL := test