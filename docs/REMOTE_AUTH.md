# Hosted MCP authentication

## Goals

1. Keep **static Bearer** (`sat_*` / product tokens) working for Codex, Claude Code, Gemini CLI, VS Code, Windsurf, Zed, and Cursor builds that accept `bearer_token_env_var`.
2. Add **first-class PipeOps OAuth 2.1** so Claude Desktop, ChatGPT custom MCP apps, Cursor OAuth, and Codex `mcp login` get browser login without pasting tokens.
3. Optionally ship an **MCP-only OAuth bridge** (token paste → short-lived MCP tokens) behind a flag if Controller/Console work is delayed.

## What the MCP server owns today (Phase 1)

Hosted process: official Go MCP SDK, **stateless Streamable HTTP**.

| Surface | Behavior |
|---------|----------|
| `POST` / `GET` / `DELETE /mcp` | Streamable HTTP; **Bearer required** |
| Token validation | Online via PipeOps `GET /profile/data` |
| Credential isolation | Fresh SDK client per request (token never shared across callers) |
| `GET /.well-known/oauth-protected-resource` | Resource metadata |
| `GET /.well-known/oauth-protected-resource/mcp` | Path-scoped resource metadata |
| Unauthenticated `/mcp` | `401` + `WWW-Authenticate: Bearer resource_metadata="…"` |
| `GET /healthz` | Unauthenticated health |
| STDIO | Local dev / CI with `PIPEOPS_TOKEN` |

Phase 1 accepts existing workspace service tokens (`sat_…`) and user product tokens and **forwards** them to the controller. Audience-bound MCP tokens + exchange are Phase 2c.

### Critical discovery rule

**Do not advertise an authorization server until that issuer serves complete AS metadata.**

| Env | Effect |
|-----|--------|
| `PIPEOPS_OAUTH_ISSUER` **unset** (default) | `authorization_servers` is **empty**. Clients must use static Bearer. No broken browser login. |
| `PIPEOPS_OAUTH_ISSUER=https://api.pipeops.io` | Advertise that issuer **only after** it serves `GET /.well-known/oauth-authorization-server` and live authorize/token endpoints. |

Previously the server defaulted issuer to `https://api.pipeops.io` without AS discovery, so OAuth clients started a flow that could not finish. That default is removed.

## Deploying the MCP service (Bearer available now)

```text
PIPEOPS_TRANSPORT=http
PIPEOPS_HTTP_ADDR=:8080
PIPEOPS_BASE_URL=https://api.pipeops.io
PIPEOPS_MCP_PUBLIC_URL=https://mcp.pipeops.app/mcp
# PIPEOPS_OAUTH_ISSUER=   # leave unset until first-class OAuth AS is ready
```

Route `https://mcp.pipeops.app` → port 8080 (TLS at ingress). **Do not** set a shared `PIPEOPS_TOKEN` in hosted mode.

### Customer setup — available now (Bearer)

```bash
export PIPEOPS_TOKEN="sat_your_workspace_token"
codex mcp add pipeops \
  --url https://mcp.pipeops.app/mcp \
  --bearer-token-env-var PIPEOPS_TOKEN
```

Compatible with any client that supports remote Streamable HTTP + Bearer env.

### Customer setup — after first-class OAuth

```bash
codex mcp add pipeops --url https://mcp.pipeops.app/mcp
codex mcp login pipeops
```

(And equivalent Claude / Cursor / ChatGPT connector OAuth UIs.)

## Path 1 — First-class PipeOps OAuth (recommended)

Reuse the **existing** controller OAuth AS (CLI already uses `/oauth/authorize` + `/oauth/token` + PKCE). Do **not** build a second AS.

### Controller (AS)

| Item | Status / work |
|------|----------------|
| `GET /.well-known/oauth-authorization-server` | Required before setting `PIPEOPS_OAUTH_ISSUER` |
| `GET /oauth/authorize` | Exists (session JWT) |
| `POST /oauth/token` | Exists (auth code + PKCE) |
| PKCE S256 | Exists |
| Refresh-token grant + rotation | Confirm/add before advertising `refresh_token` |
| Dynamic client registration (`POST /oauth/register`) | Optional; or pre-register Claude/Cursor/Codex public clients |
| Consent + workspace binding | Auto-approve today → explicit consent + `workspace_ids` |
| Audience `aud=https://mcp.pipeops.app/mcp` | Required for MCP tokens |
| Scopes | `pipeops:read`, `projects:write`, `deployments:write`, `addons:write`, `billing:write`, `tokens:admin` |
| SA denylist / secret export | Preserve for exchanged credentials |

### Dashboard (console)

1. Browser front door for authorize (preserve `state`, PKCE, scopes, resource, redirect).
2. Consent UI: app name, scope groups, **workspace selection**, approve/deny.
3. Redirect to client callback with authorization code.

### MCP (after AS is live)

1. Set `PIPEOPS_OAUTH_ISSUER` to the real issuer (API base that serves AS metadata).
2. Accept MCP access tokens (validate via introspect/JWT + audience).
3. **Phase 2c:** exchange MCP token → short-lived controller credential (no passthrough of long-lived product tokens).

## Path 2 — MCP-only OAuth bridge (interim, feature-flagged)

If Controller/Console cannot ship yet, MCP can host a **temporary** AS on `mcp.pipeops.app`:

1. Client runs OAuth against MCP.
2. Secure page: customer pastes existing workspace service token.
3. MCP validates token against PipeOps.
4. MCP stores encrypted upstream token; issues short-lived audience-bound access + refresh to the AI client.
5. Revocation deletes grant + stored credential.

**Trade-offs:** still requires creating/pasting `sat_*`; MCP becomes a secret vault. Treat as bridge only (`PIPEOPS_MCP_OAUTH_BRIDGE=true`). Prefer Path 1.

## Client matrix

| Client | Available now (Bearer) | After PipeOps OAuth | Notes |
|--------|------------------------|---------------------|--------|
| Codex | Yes (`--bearer-token-env-var`) | `mcp login` | Bearer works today |
| Claude Code / Gemini CLI / VS Code / Windsurf / Zed | Yes (Bearer env) | Yes when they use OAuth | |
| Claude Desktop connectors | Limited | Yes | Often OAuth-only |
| Cursor (Bearer-capable builds) | Yes | Yes | |
| Cursor OAuth flow | No | Yes | Needs Path 1 or bridge |
| ChatGPT custom MCP apps | No | Yes | Needs Path 1 or bridge |
| Other RFC-compliant remote MCP | Bearer if supported | Yes | |

## Acceptance checks

### Phase 1 (Bearer)

```bash
curl -i https://mcp.pipeops.app/.well-known/oauth-protected-resource
# authorization_servers must be [] or omitted until AS is ready

curl -i https://mcp.pipeops.app/healthz
curl -i -X POST https://mcp.pipeops.app/mcp
# 401 + WWW-Authenticate resource_metadata=...

curl -i -X POST https://mcp.pipeops.app/mcp \
  -H "Authorization: Bearer $PIPEOPS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"curl","version":"0"}}}'
```

Codex should report **Auth: Bearer token**, not Unsupported.

### Before enabling OAuth issuer advertisement

```bash
curl -i https://api.pipeops.io/.well-known/oauth-authorization-server
# must return valid AS metadata with authorize + token endpoints
```

Only then:

```text
PIPEOPS_OAUTH_ISSUER=https://api.pipeops.io
```

## Related docs

- Controller handoff checklist: `pipeops-controller/docs/MCP_OAUTH_CONTROLLER_DELTA.md` (when present on develop)
- Service tokens / dual-auth denylist: controller `docs/service-tokens.md`
