# Publishing to MCP Registry

This document explains how to publish the PipeOps MCP server to the official Model Context Protocol registry.

## Registry Information

The MCP registry is maintained at: https://github.com/modelcontextprotocol/registry

## Prerequisites

1. The repository must contain a valid `mcp.json` file (already included)
2. The server must be publicly accessible and installable
3. The repository must have a clear README with usage instructions

## How to Submit

To publish this server to the MCP registry:

1. Fork the registry repository: https://github.com/modelcontextprotocol/registry

2. Add an entry to the `servers.json` file in the registry:

```json
{
  "name": "pipeops",
  "repository": "https://github.com/PipeOpsHQ/pipeops-mcp"
}
```

3. Create a Pull Request with:
   - Title: "Add PipeOps MCP Server"
   - Description: Brief overview of the server capabilities
   - Reference to this repository

4. Wait for the registry maintainers to review and merge

## Validation

The `mcp.json` file in this repository follows the official schema and includes:

- Server metadata (name, description, category)
- Installation instructions (Go install command)
- Configuration requirements (environment variables)
- Feature list

## After Publication

Once published, users can discover and install the server through:
- MCP registry website
- Claude Desktop integration
- Direct GitHub discovery

## Updates

To update registry information:
1. Update `mcp.json` in this repository
2. Tag a new release
3. The registry will automatically pick up changes (or submit a PR to update)
