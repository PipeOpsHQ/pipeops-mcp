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

### Hosted MCP (recommended for customers)

Customers do not install a Go binary. Add the hosted endpoint to any remote MCP
client:

```text
https://mcp.pipeops.app/mcp
```

When the hosted service runs with `PIPEOPS_OAUTH_MODE=bridge`, OAuth-capable
clients discover the authorization server automatically. The PipeOps consent
flow opens the PipeOps Console, where the customer signs in with the same PKCE
OAuth flow as `pipeops login`. The bridge returns short-lived, scoped OAuth
credentials to the MCP client. The Console access and refresh credentials are
encrypted at rest and are never returned to the client.

Clients that support static request headers can continue to connect directly:

```bash
export PIPEOPS_TOKEN="sat_your_token_here"
codex mcp add pipeops \
  --url https://mcp.pipeops.app/mcp \
  --bearer-token-env-var PIPEOPS_TOKEN
codex mcp get pipeops
```

See [`docs/REMOTE_AUTH.md`](docs/REMOTE_AUTH.md) for deployment modes, OAuth
endpoints, storage requirements, and acceptance checks.

### Local binary

```bash
go install github.com/PipeOpsHQ/pipeops-mcp/cmd/pipeops-mcp-server@latest
```

The binary is installed at `$(go env GOPATH)/bin/pipeops-mcp-server`.

To install from source instead:

```bash
git clone https://github.com/PipeOpsHQ/pipeops-mcp.git
cd pipeops-mcp
go install ./cmd/pipeops-mcp-server
```

## Configuration

The server requires authentication via either:

1. **Service / API Token (recommended)**: Set `PIPEOPS_TOKEN` to a dedicated workspace service token (`sat_…`) with `api:read` and only the additional scopes it needs (Integrations → Service Tokens)
2. **Email/Password**: Set `PIPEOPS_EMAIL` and `PIPEOPS_PASSWORD` environment variables

Optional configuration:
- `PIPEOPS_BASE_URL`: PipeOps API base URL (default: https://api.pipeops.io)

Hosted transport configuration:

- `PIPEOPS_TRANSPORT=http` enables Streamable HTTP (default remains `stdio`)
- `PIPEOPS_HTTP_ADDR=:8080` controls the listen address
- `PIPEOPS_MCP_PUBLIC_URL=https://mcp.pipeops.app/mcp` sets the OAuth resource URL
- `PIPEOPS_OAUTH_MODE=bearer|bridge|external` selects the hosted authentication mode
- `PIPEOPS_OAUTH_STORE=sqlite|redis` selects bridge persistence (defaults to `sqlite`)
- `PIPEOPS_OAUTH_SQLITE_PATH=/data/oauth/pipeops-mcp-oauth.db` sets the SQLite file used by a single MCP replica
- `PIPEOPS_OAUTH_REDIS_URL` is required only when `PIPEOPS_OAUTH_STORE=redis`
- `PIPEOPS_OAUTH_ENCRYPTION_KEY` is required in `bridge` mode and must be a base64-encoded 32-byte key
- `PIPEOPS_OAUTH_ISSUER` optionally overrides the bridge issuer and is required in `external` mode
- `PIPEOPS_MCP_SCOPES` optionally overrides the comma-separated scopes (defaults to `api:read,api:write`)
- `PIPEOPS_CONSOLE_OAUTH_URL` optionally overrides the Console URL (defaults to `https://console.pipeops.io`)
- `PIPEOPS_CONSOLE_OAUTH_CLIENT_ID` optionally overrides the Console public OAuth client (defaults to `pipeops_public_client`)
- `PIPEOPS_CONSOLE_OAUTH_SCOPES` optionally overrides the Console scopes (defaults to `openid,profile,email`)

Generate the bridge encryption key with `openssl rand -base64 32`. Mount a
persistent volume at `/data` when using SQLite so OAuth connections survive
container replacement. Use Redis only when multiple MCP replicas need shared
state. The mounted directory must be writable by UID/GID `65532`; the server
keeps `/data/oauth` private with mode `0700` and its database files at mode
`0600`. Do not set a shared `PIPEOPS_TOKEN` on the hosted service; each customer
authorizes their own PipeOps Console session.

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

### With Codex

Set your workspace service token and register the installed server:

```bash
export PIPEOPS_TOKEN="sat_your_token_here"
codex mcp add pipeops \
  --env PIPEOPS_TOKEN="$PIPEOPS_TOKEN" \
  -- "$(go env GOPATH)/bin/pipeops-mcp-server"
codex mcp list
```

Start a new Codex task after registering the server so its tools are loaded.

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
- `restart_project` - Restart a project
- `stop_project` - Stop a project
- `get_project_logs` - Retrieve project logs
- `get_project_env_variables` - Retrieve project environment variables
- `update_project_env_variables` - Update project environment variables
- `update_project_deploy_settings` - Update deploy/source-control settings (prefer-client thin body)
- `update_project_security_policy` - Update image-scan security policy (prefer-client partial update)
- `deploy_project_from_image` - Create and deploy a project from a pre-built container image

### BYOI & Registries
- `create_external_registry` - Create an external container registry configuration
- `list_external_registries` - List external registries for a workspace
- `get_external_registry` - Get external registry details
- `delete_external_registry` - Delete an external registry configuration
- `list_external_registry_images` - List repositories available through an external registry
- `list_external_registry_tags` - List image tags available through an external registry
- `search_public_registry_images` - Search public Docker Hub images
- `list_public_registry_tags` - List tags for a public Docker Hub image

### Servers
- `list_servers` - List all servers
- `get_server` - Get server details
- `get_cluster_connection` - Get cluster connection information (`server_id` also works as an alias)
- `get_cluster_cost_allocation` - Get cluster compute cost allocation (`server_id` also works as an alias)

### Cloud Providers
- `list_cloud_provider_regions` - List available regions for a cloud provider
- `list_cloud_provider_instance_categories` - List available instance categories for a cloud provider
- `list_cloud_provider_instance_types` - List available instance types for a region and instance class
- `list_cloud_provider_server_templates` - List recommended server templates for a cloud provider

### Environments
- `list_environments` - List environment configurations
- `get_environment` - Get environment details
- `create_environment` - Create a new environment
- `update_environment` - Update environment metadata
- `delete_environment` - Delete an environment
- `set_environment_variables` - Set environment variables for an environment

### Teams & Workspaces
- `list_teams` - List teams
- `create_team` - Create a team
- `update_team` - Update a team
- `get_team` - Get team details
- `invite_team_member` - Invite a team member
- `list_team_members` - List members of a team
- `remove_team_member` - Remove a team member
- `update_team_member_role` - Update a team member role
- `list_workspaces` - List workspaces
- `create_workspace` - Create a workspace
- `update_workspace` - Update a workspace
- `get_workspace` - Get workspace details
- `delete_workspace` - Delete a workspace
- `set_workspace_billing_email` - Set workspace billing email

### Billing
- `get_billing_info` - Get billing information
- `list_billing_plans` - List available billing plans
- `subscribe_to_plan` - Subscribe to a billing plan
- `cancel_subscription` - Cancel a subscription
- `add_billing_card` - Add a billing card
- `delete_billing_card` - Delete a billing card
- `create_workspace_checkout` - Create workspace billing or checkout configuration
- `start_trial` - Start a billing trial
- `get_billing_portal_url` - Get the billing portal URL
- `get_balance` - Get current account balance
- `list_workspace_cards` - List current workspace billing cards
- `get_active_card` - Get the active billing card
- `list_subscriptions` - List subscriptions
- `get_subscription` - Get subscription details
- `list_invoices` - List invoices

### Security
- `list_service_account_tokens` - List service account tokens
- `get_service_account_token` - Get service account token details
- `create_service_account_token` - Create a service account token
- `update_service_account_token` - Update a service account token
- `revoke_service_account_token` - Revoke a service account token

### Users
- `get_current_user` - Get current user profile

### Add-ons
- `list_addons` - List available add-ons
- `get_addon` - Get add-on details
- `deploy_addon` - Deploy an add-on to a project or server
- `list_addon_deployments` - List add-on deployments
- `get_addon_deployment` - Get add-on deployment details
- `get_addon_deployment_session` - Get add-on deployment session details
- `view_addon_deployment_configs` - View add-on deployment config
- `add_addon_domain` - Add a domain to an add-on
- `list_addon_categories` - List add-on categories
- `get_my_addon_submissions` - List your add-on submissions

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
    "project_id": "proj_abc123",
    "workspace_id": "workspace_abc123",
    "no_cache": false
  }
}
```

### Update Project Environment Variables
```json
{
	"name": "update_project_env_variables",
	"arguments": {
		"project_id": "proj_abc123",
		"env_variables": [
			{
				"key": "DATABASE_URL",
				"value": "postgres://..."
			},
			{
				"key": "API_KEY",
				"value": "secret123"
			}
		]
	}
}
```

### Deploy From an Image
```json
{
	"name": "deploy_project_from_image",
	"arguments": {
		"name": "nginx-demo",
		"container_image": "docker.io/library/nginx",
		"image_tag": "latest",
		"port": 80,
		"vcpu": 0.5,
		"memory": {
			"value": 512,
			"unit": "MB"
		},
		"server_id": "cluster_abc123",
		"environment_id": "env_abc123",
		"workspace_id": "ws_abc123"
	}
}
```

### Deploy an Add-on
```json
{
	"name": "deploy_addon",
	"arguments": {
		"addon_id": "addon_abc123",
		"project_id": "proj_abc123",
		"config": {
			"plan": "starter"
		}
	}
}
```

### Start a Trial
```json
{
	"name": "start_trial",
	"arguments": {
		"plan_id": "growth_v1"
	}
}
```

## Development

```bash
# Clone repository
git clone https://github.com/PipeOpsHQ/pipeops-mcp
cd pipeops-mcp

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
