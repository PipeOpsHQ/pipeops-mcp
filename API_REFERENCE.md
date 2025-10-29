# PipeOps MCP Server API Reference

Complete reference for all available MCP tools provided by the PipeOps MCP Server.

## Table of Contents

- [Projects](#projects)
- [Servers](#servers)
- [Environments](#environments)
- [Teams](#teams)
- [Workspaces](#workspaces)
- [Users](#users)
- [Billing](#billing)
- [Add-ons](#add-ons)

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

## Environments

### list_environments

List all environments in your account.

**Arguments:**
- `project_id` (string, required): The project ID (currently not used by API but required for future compatibility)

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

Get current billing usage and information.

**Arguments:** None

**Example:**
```
What's my current PipeOps usage?
```
or
```
Show me my billing information
```

**Response:**
```json
{
  "status": "success",
  "message": "Usage retrieved successfully",
  "data": {
    "usage": [
      {
        "resource_type": "compute",
        "amount": 120.5,
        "unit": "hours",
        "cost": 24.10,
        "period": "2024-10"
      },
      {
        "resource_type": "storage",
        "amount": 50.0,
        "unit": "GB",
        "cost": 5.00,
        "period": "2024-10"
      }
    ],
    "total": 29.10
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
