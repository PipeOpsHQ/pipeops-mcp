# Contributing to PipeOps MCP Server

Thank you for your interest in contributing to the PipeOps MCP Server!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/pipeops-mcp.git`
3. Create a feature branch: `git checkout -b feature/my-new-feature`
4. Make your changes
5. Run tests: `make test`
6. Commit your changes: `git commit -am 'Add some feature'`
7. Push to the branch: `git push origin feature/my-new-feature`
8. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.21 or higher
- PipeOps account with API access
- Make (optional, but recommended)

### Installation

```bash
# Clone the repository
git clone https://github.com/PipeOpsHQ/pipeops-mcp.git
cd pipeops-mcp

# Install dependencies
go mod download

# Build the project
make build

# Run tests
make test
```

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Run `make lint` to check for issues
- Write tests for new features
- Update documentation for API changes

## Adding New Tools

To add a new MCP tool:

1. Add the tool definition to `handleToolsList()` in `internal/mcp/server.go`
2. Implement the tool handler in `handleToolsCall()`
3. Create a method to interact with the PipeOps SDK
4. Update the README with the new tool documentation
5. Add tests for the new tool

Example:
```go
// In handleToolsList()
{
    Name:        "my_new_tool",
    Description: "Description of what this tool does",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param_name": map[string]string{
                "type":        "string",
                "description": "Parameter description",
            },
        },
        "required": []string{"param_name"},
    },
}

// In handleToolsCall()
case "my_new_tool":
    param, ok := callParams.Arguments["param_name"].(string)
    if !ok {
        return nil, fmt.Errorf("param_name is required")
    }
    return s.myNewTool(ctx, param)

// Implementation
func (s *Server) myNewTool(ctx context.Context, param string) (interface{}, error) {
    resp, _, err := s.client.SomeService.SomeMethod(ctx, param)
    if err != nil {
        return nil, err
    }
    return map[string]interface{}{
        "content": []interface{}{
            map[string]interface{}{
                "type": "text",
                "text": formatJSON(resp),
            },
        },
    }, nil
}
```

## Testing

- Write unit tests for new functionality
- Ensure all tests pass before submitting PR
- Add integration tests where appropriate

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test -v ./internal/mcp -run TestSpecificFunction
```

## Documentation

- Update README.md for user-facing changes
- Add godoc comments for exported functions
- Update CHANGELOG.md with notable changes

## Commit Messages

Follow conventional commit format:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test additions or changes
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

Example: `feat: add support for listing cloud providers`

## Pull Request Process

1. Update the README.md with details of changes if needed
2. Update the CHANGELOG.md with notable changes
3. Ensure all tests pass
4. Request review from maintainers
5. Address review feedback
6. Once approved, your PR will be merged

## Questions?

Feel free to open an issue for:
- Bug reports
- Feature requests
- Questions about the codebase
- General discussion

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
