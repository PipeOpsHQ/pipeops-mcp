# Quick Start Guide

Get up and running with PipeOps MCP Server in under 5 minutes!

## Step 1: Install

```bash
go install github.com/PipeOpsHQ/pipeops-mcp/cmd/pipeops-mcp-server@latest
```

The binary is installed at `$(go env GOPATH)/bin/pipeops-mcp-server`.

## Step 2: Get Your API Token

1. Visit [PipeOps Dashboard](https://pipeops.io)
2. Go to **Settings** → **API Tokens**
3. Click **Create New Token**
4. Copy your token (save it securely!)

## Step 3: Configure Claude Desktop

### macOS / Linux

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "pipeops": {
      "command": "pipeops-mcp-server",
      "env": {
        "PIPEOPS_TOKEN": "paste-your-token-here"
      }
    }
  }
}
```

### Windows

Edit `%APPDATA%\Claude\claude_desktop_config.json` with the same content.

## Step 4: Restart Claude Desktop

Close and reopen Claude Desktop completely.

## Step 5: Test It!

In Claude, try these commands:

```
List all my PipeOps projects
```

```
Show me my current billing usage
```

```
What servers do I have?
```

## What's Next?

- 📖 Read the [Full Documentation](README.md)
- 🔧 See [Setup Guide](SETUP.md) for advanced configuration
- 📚 Browse [API Reference](API_REFERENCE.md) for all available tools
- 🤝 Check [Contributing Guide](CONTRIBUTING.md) to contribute

## Troubleshooting

### "Command not found"
- Verify installation: `which pipeops-mcp-server`
- Use full path in config if needed

### Authentication errors
- Double-check your token
- Ensure no extra spaces in the config
- Verify token hasn't expired

### No response from Claude
- Check Claude's MCP logs
- Restart Claude Desktop
- Verify the config file syntax is valid JSON

## Need Help?

- 🐛 [Report Issues](https://github.com/PipeOpsHQ/pipeops-mcp/issues)
- 💬 [Discussions](https://github.com/PipeOpsHQ/pipeops-mcp/discussions)
- 📧 support@pipeops.io

---

**Happy deploying! 🚀**
