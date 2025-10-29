.PHONY: build test clean install run lint

# Build the binary
build:
	go build -o pipeops-mcp-server ./cmd/pipeops-mcp-server

# Install the binary
install:
	go install ./cmd/pipeops-mcp-server

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -f pipeops-mcp-server
	rm -f coverage.out

# Run the server
run: build
	./pipeops-mcp-server

# Lint the code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o dist/pipeops-mcp-server-linux-amd64 ./cmd/pipeops-mcp-server
	GOOS=darwin GOARCH=amd64 go build -o dist/pipeops-mcp-server-darwin-amd64 ./cmd/pipeops-mcp-server
	GOOS=darwin GOARCH=arm64 go build -o dist/pipeops-mcp-server-darwin-arm64 ./cmd/pipeops-mcp-server
	GOOS=windows GOARCH=amd64 go build -o dist/pipeops-mcp-server-windows-amd64.exe ./cmd/pipeops-mcp-server
