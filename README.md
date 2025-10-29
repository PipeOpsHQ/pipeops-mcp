# PipeOps MCP Server

Model Context Protocol (MCP) server for interacting with PipeOps platform.

## Features

- **Projects**: List, get, create, update, and delete projects
- **Servers**: Manage servers and deployments
- **Environments**: Handle environment configurations
- **Teams**: Team and workspace management
- **Billing**: View billing information and usage
- **Add-ons**: Manage add-ons and integrations
- **Users**: User profile and authentication
- **Cloud Providers**: Configure cloud provider integrations

## Installation

```bash
go install github.com/PipeOpsHQ/pipeops-mcp-server/cmd/pipeops-mcp-server@latest
```

## Configuration

The server requires authentication via either:

1. **Email/Password**: Set `PIPEOPS_EMAIL` and `PIPEOPS_PASSWORD` environment variables
2. **API Token**: Set `PIPEOPS_TOKEN` environment variable

Optional configuration:
- `PIPEOPS_BASE_URL`: PipeOps API base URL (default: https://api.pipeops.io)

## Usage

### With Claude Desktop

Add to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

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

### Standalone

```bash
export PIPEOPS_TOKEN="your-api-token"
pipeops-mcp-server
```

## Available Tools

### Projects
- `list_projects` - List all projects with optional filtering
- `get_project` - Get detailed project information
- `create_project` - Create a new project
- `update_project` - Update project configuration
- `delete_project` - Delete a project
- `deploy_project` - Trigger project deployment

### Servers
- `list_servers` - List all servers
- `get_server` - Get server details
- `create_server` - Provision a new server
- `update_server` - Update server configuration
- `delete_server` - Delete a server
- `restart_server` - Restart a server

### Environments
- `list_environments` - List environment configurations
- `get_environment` - Get environment details
- `create_environment` - Create new environment
- `update_environment` - Update environment variables
- `delete_environment` - Delete environment

### Teams & Workspaces
- `list_teams` - List teams
- `get_team` - Get team details
- `list_workspaces` - List workspaces
- `get_workspace` - Get workspace details

### Billing
- `get_billing_info` - Get billing information
- `get_usage` - Get resource usage statistics

### Users
- `get_current_user` - Get current user profile
- `update_user_profile` - Update user profile

## Examples

### List Projects
```json
{
  "name": "list_projects",
  "arguments": {}
}
```

### Deploy a Project
```json
{
  "name": "deploy_project",
  "arguments": {
    "project_id": "proj_abc123"
  }
}
```

### Create Environment Variable
```json
{
  "name": "update_environment",
  "arguments": {
    "project_id": "proj_abc123",
    "environment": "production",
    "variables": {
      "DATABASE_URL": "postgres://...",
      "API_KEY": "secret123"
    }
  }
}
```

## Development

```bash
# Clone repository
git clone https://github.com/PipeOpsHQ/pipeops-mcp-server
cd pipeops-mcp-server

# Install dependencies
go mod download

# Run tests
go test ./...

# Build
go build -o pipeops-mcp-server ./cmd/pipeops-mcp-server

# Run locally
./pipeops-mcp-server
```

## License

MIT License

## Contributing

Contributions welcome! Please open an issue or submit a pull request.
# pipeops-mcp
