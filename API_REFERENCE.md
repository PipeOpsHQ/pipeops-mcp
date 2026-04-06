# PipeOps MCP Server API Reference

High-level reference for the currently implemented MCP tools provided by the PipeOps MCP Server.
Use `tools/list` as the canonical source for the exact live tool inventory and schemas.

## Table of Contents

- [Current Tool Inventory](#current-tool-inventory)
- [Projects](#projects)
- [BYOI & Registries](#byoi--registries)
- [Servers](#servers)
- [Cloud Providers](#cloud-providers)
- [Environments](#environments)
- [Teams](#teams)
- [Workspaces](#workspaces)
- [Users](#users)
- [Billing](#billing)
- [Add-ons](#add-ons)

---

## Current Tool Inventory

### Projects

- `list_projects`
- `get_project`
- `create_project`
- `update_project`
- `delete_project`
- `deploy_project`
- `restart_project`
- `stop_project`
- `get_project_logs`
- `get_project_env_variables`
- `update_project_env_variables`
- `deploy_project_from_image`

### BYOI & Registries

- `create_external_registry`
- `list_external_registries`
- `get_external_registry`
- `delete_external_registry`
- `list_external_registry_images`
- `list_external_registry_tags`
- `search_public_registry_images`
- `list_public_registry_tags`

### BYOI & Registries

- `deploy_project_from_image` creates a new project directly from a container image.
- `create_external_registry`, `list_external_registries`, `get_external_registry`, and `delete_external_registry` manage workspace-scoped registry credentials.
- `list_external_registry_images` and `list_external_registry_tags` browse authenticated Docker Hub content.
- `search_public_registry_images` and `list_public_registry_tags` browse public Docker Hub content without saving credentials.

---

## Servers

- `list_servers`
- `get_server`
- `get_cluster_connection`
- `get_cluster_cost_allocation`

### Cloud Providers

- `list_cloud_provider_regions`
- `list_cloud_provider_instance_categories`
- `list_cloud_provider_instance_types`
- `list_cloud_provider_server_templates`

### Cloud Providers

- `list_cloud_provider_regions` lists available regions for a provider like `aws`, `gcp`, `huawei`, or `digital_ocean`.
- `list_cloud_provider_instance_categories` lists available instance families for a provider.
- `list_cloud_provider_instance_types` lists concrete machine sizes for an instance class and region.
- `list_cloud_provider_server_templates` lists recommended server templates grouped by environment.

---

## Environments

- `list_environments`
- `get_environment`
- `create_environment`
- `update_environment`
- `delete_environment`
- `set_environment_variables`

### Teams

- `list_teams`
- `create_team`
- `update_team`
- `get_team`
- `invite_team_member`
- `list_team_members`
- `remove_team_member`
- `update_team_member_role`

### Workspaces

- `list_workspaces`
- `create_workspace`
- `update_workspace`
- `get_workspace`
- `delete_workspace`
- `set_workspace_billing_email`

### Users

- `get_current_user`

### Billing

- `get_billing_info`
- `list_billing_plans`
- `subscribe_to_plan`
- `cancel_subscription`
- `add_billing_card`
- `delete_billing_card`
- `create_workspace_checkout`
- `start_trial`
- `get_billing_portal_url`
- `get_balance`
- `list_workspace_cards`
- `get_active_card`
- `list_subscriptions`
- `get_subscription`
- `list_invoices`

### Add-ons

- `list_addons`
- `get_addon`
- `deploy_addon`
- `list_addon_deployments`
- `get_addon_deployment`
- `get_addon_deployment_session`
- `view_addon_deployment_configs`
- `add_addon_domain`
- `list_addon_categories`
- `get_my_addon_submissions`

### Security

- `list_service_account_tokens`
- `get_service_account_token`
- `create_service_account_token`
- `update_service_account_token`
- `revoke_service_account_token`

---

## Projects

### list_projects

List all projects in your PipeOps account.

**Arguments:** None

**Example:**
```
List all my projects
```

**Response:**
```json
{
  "status": "success",
  "message": "Projects retrieved successfully",
  "data": {
    "projects": [
      {
        "id": "proj_123",
        "uuid": "uuid-123",
        "name": "My API",
        "status": "running",
        "repository": "github.com/user/repo",
        "branch": "main"
      }
    ]
  }
}
```

---

### get_project

Get detailed information about a specific project.

**Arguments:**
- `project_id` (string, required): The project ID or UUID

**Example:**
```
Get details for project proj_abc123
```

**Response:**
```json
{
  "status": "success",
  "message": "Project retrieved successfully",
  "data": {
    "project": {
      "id": "proj_abc123",
      "name": "My API",
      "description": "Production API",
      "status": "running",
      "server_id": "srv_xyz",
      "environment_id": "env_123",
      "repository": "github.com/user/repo",
      "branch": "main",
      "build_command": "npm run build",
      "start_command": "npm start",
      "port": 3000,
      "framework": "nodejs",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T00:00:00Z"
    }
  }
}
```

---

### deploy_project

Trigger a deployment for a project.

**Arguments:**
- `project_id` (string, required): The project ID to deploy

**Example:**
```
Deploy project proj_abc123
```

**Response:**
```
Deployment triggered successfully
```

---

## Servers

### list_servers

List all servers/clusters in your account.

**Arguments:** None

**Example:**
```
Show me all my servers
```

**Response:**
```json
{
  "status": "success",
  "message": "Servers retrieved successfully",
  "data": {
    "servers": [
      {
        "id": "srv_123",
        "uuid": "uuid-456",
        "name": "Production Server",
        "provider": "aws",
        "region": "us-east-1",
        "status": "active",
        "workspace_id": "wks_789",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

### get_server

Get detailed information about a specific server.

**Arguments:**
- `server_id` (string, required): The server ID or UUID

**Example:**
```
Get details for server srv_xyz789
```

**Response:**
```json
{
  "status": "success",
  "message": "Server retrieved successfully",
  "data": {
    "server": {
      "id": "srv_xyz789",
      "uuid": "uuid-xyz",
      "name": "Production Server",
      "provider": "aws",
      "region": "us-east-1",
      "status": "active",
      "workspace_id": "wks_789",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T00:00:00Z"
    }
  }
}
```

---

### get_cluster_connection

Get cluster connection details, including connection metadata returned by the controller.

**Arguments:**
- `cluster_id` (string, required): The cluster ID or UUID
- `server_id` (string, optional): Alias for `cluster_id`

**Example:**
```
Show me how to connect to cluster cls_123
```

**Response:**
```json
{
  "status": "success",
  "message": "Cluster connection retrieved successfully",
  "data": {
    "connection": {
      "host": "cluster.example.pipeops.io",
      "port": 443,
      "kubeconfig": "..."
    }
  }
}
```

---

### get_cluster_cost_allocation

Get compute cost allocation details for a cluster.

**Arguments:**
- `cluster_id` (string, required): The cluster ID or UUID
- `server_id` (string, optional): Alias for `cluster_id`

**Example:**
```
Show me cost allocation for cluster cls_123
```

**Response:**
```json
{
  "status": "success",
  "message": "Cluster cost allocation retrieved successfully",
  "data": {
    "costs": {
      "cpu": 12.34,
      "memory": 8.91,
      "total": 21.25
    }
  }
}
```

---

## Environments

### list_environments

List all environments in your account.

**Arguments:** None

**Example:**
```
List all environments
```

**Response:**
```json
{
  "status": "success",
  "message": "Environments retrieved successfully",
  "data": {
    "environments": [
      {
        "id": "env_123",
        "uuid": "uuid-env",
        "name": "production",
        "workspace_id": "wks_789",
        "created_at": "2024-01-01T00:00:00Z"
      },
      {
        "id": "env_456",
        "uuid": "uuid-env2",
        "name": "staging",
        "workspace_id": "wks_789",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

## Teams

### list_teams

List all teams you're a member of.

**Arguments:** None

**Example:**
```
Show me all my teams
```

**Response:**
```json
{
  "status": "success",
  "message": "Teams retrieved successfully",
  "data": {
    "teams": [
      {
        "id": "team_123",
        "uuid": "uuid-team",
        "name": "Engineering",
        "description": "Engineering team",
        "members_count": 5,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

## Workspaces

### list_workspaces

List all workspaces accessible to you.

**Arguments:** None

**Example:**
```
List all my workspaces
```

**Response:**
```json
{
  "status": "success",
  "message": "Workspaces retrieved successfully",
  "data": {
    "workspaces": [
      {
        "id": "wks_123",
        "uuid": "uuid-wks",
        "name": "Production",
        "description": "Production workspace",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

---

## Users

### get_current_user

Get the current authenticated user's profile.

**Arguments:** None

**Example:**
```
Show my profile
```
or
```
Who am I logged in as?
```

**Response:**
```json
{
  "status": "success",
  "message": "Profile retrieved successfully",
  "data": {
    "user": {
      "id": "usr_123",
      "uuid": "uuid-usr",
      "email": "user@example.com",
      "first_name": "John",
      "last_name": "Doe",
      "avatar": "https://...",
      "created_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

---

## Billing

### get_billing_info

Get current billing balance and subscription information.

**Arguments:** None

**Example:**
```
What's my current PipeOps billing status?
```
or
```
Show me my billing information
```

**Response:**
```json
{
  "success": true,
  "message": "Billing information retrieved successfully",
  "data": {
    "balance": {
      "Balance": "0.01",
      "Currency": "USD"
    },
    "current_subscription": {
      "UID": "222882c6-13d1-4e06-abff-a53a333849db",
      "PlanTier": "startup",
      "PlanName": "Start-up",
      "Amount": "34.99",
      "BillingType": "trial",
      "Status": "active"
    }
  }
}
```

---

## Add-ons

### list_addons

List all available add-ons and their status.

**Arguments:** None

**Example:**
```
Show me available add-ons
```
or
```
What add-ons can I use?
```

**Response:**
```json
{
  "status": "success",
  "message": "Add-ons retrieved successfully",
  "data": {
    "addons": [
      {
        "id": "addon_123",
        "name": "PostgreSQL",
        "description": "Managed PostgreSQL database",
        "category": "database",
        "status": "available",
        "pricing": {
          "starting_at": 10.00,
          "currency": "USD"
        }
      },
      {
        "id": "addon_456",
        "name": "Redis",
        "description": "Managed Redis cache",
        "category": "cache",
        "status": "active",
        "pricing": {
          "starting_at": 5.00,
          "currency": "USD"
        }
      }
    ]
  }
}
```

---

## Security

### list_service_account_tokens

List service account tokens for the current account or workspace context.

**Arguments:** None

**Example:**
```
List my service account tokens
```

**Response:**
```json
{
  "status": "success",
  "message": "Service account tokens retrieved successfully",
  "data": {
    "tokens": [
      {
        "uuid": "token_123",
        "name": "ci-bot",
        "description": "Token used by CI",
        "permissions": ["projects:read", "deployments:write"],
        "is_active": true
      }
    ],
    "total": 1
  }
}
```

---

### get_service_account_token

Get detailed information about a single service account token.

**Arguments:**
- `token_id` (string, required): The service account token ID or UUID

**Example:**
```
Show me details for service account token token_123
```

**Response:**
```json
{
  "status": "success",
  "message": "Service account token retrieved successfully",
  "data": {
    "token": {
      "uuid": "token_123",
      "name": "ci-bot",
      "permissions": ["projects:read", "deployments:write"],
      "is_active": true
    }
  }
}
```

---

### create_service_account_token

Create a new service account token.

**Arguments:**
- `name` (string, required): The token name
- `description` (string, optional): The token description
- `permissions` (array[string], optional): Permissions to grant
- `expires_at` (string, optional): Expiration timestamp

**Example:**
```
Create a service account token named ci-bot with deployment permissions
```

**Response:**
```json
{
  "status": "success",
  "message": "Service account token created successfully",
  "data": {
    "token": {
      "uuid": "token_456",
      "name": "ci-bot",
      "token": "po_live_xxx",
      "permissions": ["projects:read", "deployments:write"],
      "is_active": true
    }
  }
}
```

---

### update_service_account_token

Update a service account token's metadata, permissions, or active state.

**Arguments:**
- `token_id` (string, required): The service account token ID or UUID
- `name` (string, optional): Updated token name
- `description` (string, optional): Updated description
- `permissions` (array[string], optional): Updated permissions
- `is_active` (boolean, optional): Whether the token remains active

**Example:**
```
Disable service account token token_456
```

**Response:**
```json
{
  "status": "success",
  "message": "Service account token updated successfully",
  "data": {
    "token": {
      "uuid": "token_456",
      "name": "ci-bot",
      "is_active": false
    }
  }
}
```

---

### revoke_service_account_token

Revoke and delete a service account token.

**Arguments:**
- `token_id` (string, required): The service account token ID or UUID

**Example:**
```
Revoke service account token token_456
```

**Response:**
```
Service account token revoked successfully
```

---

## Error Responses

All tools may return error responses in the following format:

```json
{
  "error": {
    "code": -32000,
    "message": "Error description",
    "data": {
      "details": "Additional error information"
    }
  }
}
```

### Common Error Codes

- `-32600`: Invalid Request
- `-32601`: Method not found
- `-32602`: Invalid params
- `-32603`: Internal error
- `-32000`: Server error (includes API errors)

### Common API Errors

- `401 Unauthorized`: Invalid or expired authentication token
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

---

## Best Practices

### 1. Error Handling

Always handle potential errors gracefully:
```
Try to deploy project proj_123, and if it fails, tell me what went wrong
```

### 2. Resource Discovery

Start with list operations to discover available resources:
```
First, list all my projects, then show me details for the first one
```

### 3. Chaining Operations

You can chain multiple operations:
```
List my projects, then deploy the one named "production-api"
```

### 4. Filtering Results

Be specific about what you need:
```
Show me only running projects
```

---

## Rate Limits

The PipeOps API has rate limits. The MCP server handles retries automatically, but you may experience delays if you hit rate limits.

**Current Limits:**
- 100 requests per minute per API token
- 1000 requests per hour per API token

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/PipeOpsHQ/pipeops-mcp-server/issues
- PipeOps Documentation: https://docs.pipeops.io
- PipeOps Support: support@pipeops.io
