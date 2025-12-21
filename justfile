set windows-shell := ["C:/Program Files/Git/bin/bash.exe", "-c"]

# Default recipe to display help information
default:
    @just --list

# Run tests with coverage
test:
    go test -v -race -cover ./...

# Run linter
lint:
    golangci-lint run --fix --timeout 30m

# Modernize code to use latest Go idioms
modernize:
    go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...

# Download dependencies
deps:
    go mod download
    go mod tidy

# Clean build artifacts
clean:
    go clean -cache -testcache
