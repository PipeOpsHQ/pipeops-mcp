# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-10-29

### Added
- Initial release of PipeOps MCP Server
- Support for MCP protocol version 2024-11-05
- Project management tools:
  - `list_projects` - List all projects
  - `get_project` - Get project details
  - `deploy_project` - Deploy a project
- Server management tools:
  - `list_servers` - List all servers
  - `get_server` - Get server details
- Environment management:
  - `list_environments` - List environments
- Team and workspace tools:
  - `list_teams` - List teams
  - `list_workspaces` - List workspaces
- User profile:
  - `get_current_user` - Get current user profile
- Billing and add-ons:
  - `get_billing_info` - Get billing usage information
  - `list_addons` - List available add-ons
- Authentication via API token or email/password
- Comprehensive error handling
- Detailed logging support

### Security
- Secure credential handling via environment variables
- No credentials stored in code or configuration files

[1.0.0]: https://github.com/PipeOpsHQ/pipeops-mcp-server/releases/tag/v1.0.0
