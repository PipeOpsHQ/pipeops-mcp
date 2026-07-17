# Hosted MCP authentication

## What the MCP server owns

The hosted process runs the official Go MCP SDK over stateless Streamable HTTP.
It provides:

- `POST`, `GET`, and `DELETE /mcp` through the Streamable HTTP transport
- Bearer authentication for every `/mcp` request
- online token validation through the existing PipeOps `GET /profile/data` API
- a fresh PipeOps SDK client per request, preventing credentials from crossing callers
- `GET /.well-known/oauth-protected-resource`
- `GET /.well-known/oauth-protected-resource/mcp`
- `401` responses with an OAuth protected-resource metadata challenge
- read-only and destructive tool annotations
- unauthenticated `GET /healthz`
- the existing STDIO transport for local development and CI

Phase 1 accepts existing workspace service tokens (`sat_...`) and user product
tokens. The authenticated token is passed to the controller for tool API calls.
This is the agreed beta behavior; audience-bound OAuth tokens require the later
exchange work described below.

## Deploying the MCP service

Required runtime settings:

```text
PIPEOPS_TRANSPORT=http
PIPEOPS_HTTP_ADDR=:8080
PIPEOPS_BASE_URL=https://api.pipeops.io
PIPEOPS_MCP_PUBLIC_URL=https://mcp.pipeops.io/mcp
PIPEOPS_OAUTH_ISSUER=https://api.pipeops.io
```

Route `https://mcp.pipeops.io` to port 8080 and terminate TLS at the ingress or
load balancer. Do not set a shared `PIPEOPS_TOKEN` in hosted mode; every customer
supplies their own Bearer token. Use `/healthz` for health probes.

Customer setup for Phase 1:

```bash
export PIPEOPS_TOKEN="sat_your_token_here"
codex mcp add pipeops \
  --url https://mcp.pipeops.io/mcp \
  --bearer-token-env-var PIPEOPS_TOKEN
```

## Controller handoff (no controller code is changed here)

Phase 1 needs no controller release. Existing product and service tokens only
need to remain valid for `GET /profile/data` and the APIs their scopes permit.

For browser login with `codex mcp login pipeops`, the controller team needs to:

1. Publish `GET /.well-known/oauth-authorization-server` for the existing PipeOps OAuth issuer.
2. Pre-register a public `Codex MCP` client with PKCE S256 and Codex loopback redirect URIs, or add tightly constrained RFC 7591 dynamic registration.
3. Confirm or add refresh-token support before advertising the `refresh_token` grant.
4. Keep using the existing `/oauth/authorize` and `/oauth/token` implementation; do not create another authorization server.
5. Later, bind approved scopes, workspace IDs, client ID, audience, and expiry to issued MCP tokens.
6. For production audience isolation, add RFC 7662-style introspection and a short-lived controller-token exchange so the MCP stops forwarding the raw Codex token.

The authorization-server issuer should be the public API base that actually
serves `/oauth/token`. If a console URL is desired, proxy the same routes and
keep one stable issuer.

## Dashboard handoff (no dashboard code is changed here)

For browser OAuth, the dashboard team needs to:

1. Add a browser front door for the existing authorize request, preserving `state`, PKCE, scopes, resource, and redirect URI.
2. Redirect back to the Codex loopback callback with the authorization code.
3. Add explicit consent showing the app name, requested scope groups, and workspace selection.
4. Allow approve/deny and send the approved scopes/workspaces to the controller consent endpoint once that endpoint exists.

Until these handoff items ship, customers should use the Phase 1 Bearer-token
configuration. The MCP endpoint and its protected-resource discovery remain
compatible with the later browser flow.

## Acceptance checks

```bash
curl -i https://mcp.pipeops.io/.well-known/oauth-protected-resource
curl -i https://mcp.pipeops.io/healthz
curl -i -X POST https://mcp.pipeops.io/mcp
```

The unauthenticated MCP request must return `401` and a `WWW-Authenticate:
Bearer ... resource_metadata=...` challenge. An authenticated MCP initialize
request must succeed with an existing PipeOps token, and Codex must report
`Auth: Bearer token` rather than `Auth: Unsupported`.
