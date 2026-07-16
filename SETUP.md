# PipeOps MCP Server Setup Guide

This guide will help you set up and configure the PipeOps MCP Server for use with Claude Desktop and other MCP-compatible clients.

## Prerequisites

- Go 1.21 or higher (for building from source)
- PipeOps account with API access
- Claude Desktop (for Claude integration)

## Installation

### Option 1: Using Go install

```bash
go install github.com/PipeOpsHQ/pipeops-mcp/cmd/pipeops-mcp-server@latest
```

### Option 2: Install from source

```bash
# Clone the repository
git clone https://github.com/PipeOpsHQ/pipeops-mcp.git
cd pipeops-mcp

# Build and install
make install

# Or build locally
make build
```

## Authentication Setup

You need to authenticate with PipeOps. There are two methods:

### Method 1: Service / API Token (Recommended)

Use a **workspace service token** (`sat_…`) with platform scopes (`api:full` or `preset: platform`).

1. Log in to your PipeOps dashboard
2. Navigate to **Integrations → Service Tokens**
3. Create a token (preset **platform** / scopes `api:full`)
4. Copy the token (you won't be able to see it again)
5. Set the environment variable:

```bash
export PIPEOPS_TOKEN="sat_…"
export PIPEOPS_BASE_URL="https://api.pipeops.io"   # optional override
```

See also: controller docs `docs/service-tokens.md`.

### Method 2: Email/Password

```bash
export PIPEOPS_EMAIL="your-email@example.com"
export PIPEOPS_PASSWORD="your-password"
```

## Claude Desktop Configuration

### macOS

1. Locate your Claude Desktop config file:
   ```
   ~/Library/Application Support/Claude/claude_desktop_config.json
   ```

2. Edit the file (create it if it doesn't exist):

```json
{
  "mcpServers": {
    "pipeops": {
      "command": "pipeops-mcp-server",
      "env": {
        "PIPEOPS_TOKEN": "your-api-token-here"
      }
    }
  }
}
```

3. If installed from source, use the full path:

```json
{
  "mcpServers": {
    "pipeops": {
      "command": "/path/to/pipeops-mcp-server",
      "env": {
        "PIPEOPS_TOKEN": "your-api-token-here"
      }
    }
  }
}
```

### Windows

1. Locate your config file:
   ```
   %APPDATA%\Claude\claude_desktop_config.json
   ```

2. Use the same configuration as above, but adjust paths:

```json
{
  "mcpServers": {
    "pipeops": {
      "command": "C:\\path\\to\\pipeops-mcp-server.exe",
      "env": {
        "PIPEOPS_TOKEN": "your-api-token-here"
      }
    }
  }
}
```

### Linux

1. Locate your config file:
   ```
   ~/.config/Claude/claude_desktop_config.json
   ```

2. Use the same configuration as macOS

## Verifying Installation

1. Restart Claude Desktop
2. Look for the MCP icon in Claude
3. Try asking Claude: "List my PipeOps projects"

## Usage Examples

Once configured, you can interact with PipeOps through Claude:

### List Projects
```
Show me all my PipeOps projects
```

### Get Project Details
```
Get details for project proj_abc123
```

### Deploy a Project
```
Deploy project proj_abc123
```

### List Servers
```
Show me all my PipeOps servers
```

### Get Billing Information
```
What's my current PipeOps usage?
```

### Manage Environments
```
List environments for project proj_abc123
```

## Troubleshooting

### Command Not Found

If you get "command not found" errors:

1. Make sure the binary is in your PATH:
   ```bash
   which pipeops-mcp-server
   ```

2. Or use the absolute path in the config file

### Authentication Errors

If you get authentication errors:

1. Verify your token is correct
2. Check that the token hasn't expired
3. Ensure the token has the necessary permissions
4. Try generating a new token

### Permission Denied

On macOS/Linux, ensure the binary is executable:

```bash
chmod +x /path/to/pipeops-mcp-server
```

### Connection Issues

1. Check your internet connection
2. Verify PipeOps API is accessible:
   ```bash
   curl https://api.pipeops.io/health
   ```

3. Check if a proxy is interfering

### Debugging

Enable debug mode by checking Claude's logs:

**macOS:**
```bash
tail -f ~/Library/Logs/Claude/mcp*.log
```

**Windows:**
```
%APPDATA%\Claude\logs\
```

**Linux:**
```bash
tail -f ~/.config/Claude/logs/mcp*.log
```

## Advanced Configuration

### Custom API Base URL

If using a self-hosted or custom PipeOps instance:

```json
{
  "mcpServers": {
    "pipeops": {
      "command": "pipeops-mcp-server",
      "env": {
        "PIPEOPS_TOKEN": "your-token",
        "PIPEOPS_BASE_URL": "https://custom.pipeops.io"
      }
    }
  }
}
```

### Multiple Environments

You can configure different PipeOps environments:

```json
{
  "mcpServers": {
    "pipeops-prod": {
      "command": "pipeops-mcp-server",
      "env": {
        "PIPEOPS_TOKEN": "prod-token"
      }
    },
    "pipeops-staging": {
      "command": "pipeops-mcp-server",
      "env": {
        "PIPEOPS_TOKEN": "staging-token",
        "PIPEOPS_BASE_URL": "https://staging.pipeops.io"
      }
    }
  }
}
```

## Security Best Practices

1. **Never commit tokens**: Don't commit your API token to version control
2. **Use environment variables**: Store tokens in environment variables
3. **Rotate tokens regularly**: Generate new tokens periodically
4. **Limit token scope**: Create tokens with minimal required permissions
5. **Monitor usage**: Regularly check your PipeOps audit logs

## Getting Help

- **Documentation**: See [README.md](README.md) for detailed API documentation
- **Issues**: Report bugs on [GitHub Issues](https://github.com/PipeOpsHQ/pipeops-mcp/issues)
- **Contributing**: See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines

## Next Steps

- Read the [API Reference](README.md#available-tools) for all available tools
- Explore [examples](examples/) for common usage patterns
- Join the community for tips and support
