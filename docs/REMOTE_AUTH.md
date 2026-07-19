# Hosted MCP authentication

The hosted PipeOps MCP server supports direct Bearer tokens and a self-contained
OAuth 2.1 bridge. The bridge is owned entirely by this repository: it does not
require controller or Console changes.

## Authentication modes

Set `PIPEOPS_OAUTH_MODE` to one of these values:

| Mode | Behavior | Use case |
| --- | --- | --- |
| `bearer` | Accept existing PipeOps tokens. OAuth discovery does not advertise an authorization server. | Default and static-header clients |
| `bridge` | Serve OAuth discovery, dynamic client registration, consent, token, refresh, and revocation endpoints. Direct PipeOps Bearer tokens continue to work. | Production remote MCP clients |
| `external` | Advertise `PIPEOPS_OAUTH_ISSUER` and accept Bearer tokens validated by PipeOps. | Future native PipeOps authorization server |

`bearer` is the default. Use `bridge` to support clients such as ChatGPT,
Claude custom connectors, and Cursor that expect a standards-based remote OAuth
flow.

## Deploy bridge mode

Required settings:

```text
PIPEOPS_TRANSPORT=http
PIPEOPS_HTTP_ADDR=:8080
PIPEOPS_BASE_URL=https://api.pipeops.io
PIPEOPS_MCP_PUBLIC_URL=https://mcp.pipeops.app/mcp
PIPEOPS_OAUTH_MODE=bridge
PIPEOPS_OAUTH_STORE=sqlite
PIPEOPS_OAUTH_SQLITE_PATH=/home/nonroot/.pipeops-mcp/oauth/pipeops-mcp-oauth.db
PIPEOPS_OAUTH_ENCRYPTION_KEY=BASE64_ENCODED_32_BYTE_KEY
```

Generate the encryption key once and store it in the deployment secret manager:

```bash
openssl rand -base64 32
```

SQLite is the default and needs no separate database service. The container
image pre-creates two directories owned by UID/GID `65532`:

- `/home/nonroot/.pipeops-mcp/oauth` — **default** path (works without a volume)
- `/data/oauth` — optional durable path when a PVC is mounted with `fsGroup`

Run one MCP replica in SQLite mode; do not put the SQLite file on a shared
network filesystem. Prefer mode `0700` on the directory and `0600` on the
database/WAL files. If the configured path is not writable, the server falls
back to `$TMPDIR/pipeops-mcp-oauth/` and logs a warning.

For durable SQLite across pod restarts on Kubernetes, mount a volume and point
the env var at it (do not mount over `/home/nonroot` unless the volume is
writable by 65532):

```yaml
securityContext:
  fsGroup: 65532
  runAsUser: 65532
  runAsGroup: 65532
env:
  - name: PIPEOPS_OAUTH_SQLITE_PATH
    value: /data/oauth/pipeops-mcp-oauth.db
volumeMounts:
  - name: oauth-data
    mountPath: /data
```

OAuth material is encrypted at rest with `PIPEOPS_OAUTH_ENCRYPTION_KEY`.

For multiple MCP replicas, use shared Redis 6.2 or newer instead:

```text
PIPEOPS_OAUTH_STORE=redis
PIPEOPS_OAUTH_REDIS_URL=rediss://USER:PASSWORD@REDIS_HOST:6379/0
```

The OAuth issuer defaults to the origin of `PIPEOPS_MCP_PUBLIC_URL`, which is
`https://mcp.pipeops.app` in production. Set `PIPEOPS_OAUTH_ISSUER` only when
the public issuer is different. Route the public hostname to port 8080 and
terminate TLS at the ingress or load balancer. Use `/healthz` for health probes.

Do not configure a shared `PIPEOPS_TOKEN` on the hosted process. Each customer
authorizes their own PipeOps Console session. Bridge mode redirects to the same
PKCE Console sign-in flow used by `pipeops login`; it never asks customers to
paste a service token.

The default Console OAuth settings are:

```text
PIPEOPS_CONSOLE_OAUTH_URL=https://console.pipeops.io
PIPEOPS_CONSOLE_OAUTH_CLIENT_ID=pipeops_public_client
PIPEOPS_CONSOLE_OAUTH_SCOPES=openid,profile,email
```

Override them only for a non-production Console environment. The Console OAuth
client must permit `https://mcp.pipeops.app/oauth/pipeops/callback` as its
redirect URI in production (or the equivalent callback for a custom issuer).

## OAuth endpoints

Bridge mode serves:

- `GET /.well-known/oauth-protected-resource`
- `GET /.well-known/oauth-protected-resource/mcp`
- `GET /.well-known/oauth-authorization-server`
- `GET /.well-known/openid-configuration`
- `POST /oauth/register`
- `GET|POST /oauth/authorize`
- `POST /oauth/token`
- `POST /oauth/revoke`
- `GET /oauth/jwks`

It implements authorization code with PKCE S256, dynamic client registration,
short-lived access tokens, refresh-token rotation with family reuse detection,
and token revocation. OAuth
tokens are opaque; `/oauth/jwks` therefore returns an empty key set for clients
that require the metadata field.

## Credential and scope model

The bridge starts a separate authorization-code-with-PKCE flow against the
PipeOps Console after an MCP client begins authorization. The Console callback
is bound to the original MCP authorization request and can be used once. The
bridge exchanges the Console code server-side, validates the resulting PipeOps
user credential with `GET /profile/data`, then encrypts the access and refresh
credentials using AES-256-GCM before writing them to the configured store.
OAuth access, refresh, and authorization codes are stored under SHA-256 lookup
keys, so their plaintext values are not used as database keys.

The bridge exposes two scopes:

- `api:read` exposes only tools annotated as read-only.
- `api:write` exposes read and mutating tools.

The upstream PipeOps user session remains the final authority. An OAuth grant
cannot grant tools outside its MCP scope, and expired Console credentials require
the customer to authorize again. Direct Bearer-token clients keep the existing
tool surface and are authorized by the controller.

Current lifetimes are 5 minutes for authorization codes, 15 minutes for access
tokens, and 30 days for refresh tokens. Authorization codes are single-use and
refresh tokens rotate on every use. Reusing an already-rotated refresh token
revokes the active refresh token in that authorization family.

## Security and operations

- Use TLS for the public MCP endpoint and for Redis when selected.
- Keep the SQLite database on a private persistent volume. When using Redis,
  restrict network access to MCP instances and enable authentication.
- Back up the encryption key in the deployment secret manager. Rotating it
  immediately invalidates existing encrypted grants unless a migration is run.
- Apply distributed rate limiting at the ingress. The process also has a local
  per-instance limiter for registration, authorization, and token requests.
- Do not log authorization headers, Console access or refresh tokens, OAuth
  codes, or request bodies from authorization and token endpoints.
- Monitor `401` and `503` rates. Missing/expired credentials fail with `401`;
  persistence or PipeOps validation failures fail closed with `503`.
- Customers can disconnect an AI client by revoking its bridge OAuth token;
  ending or revoking their PipeOps session also stops upstream authorization.

## Acceptance checks

After deployment, confirm discovery and authentication:

```bash
curl -fsS https://mcp.pipeops.app/healthz
curl -fsS https://mcp.pipeops.app/.well-known/oauth-protected-resource | jq
curl -fsS https://mcp.pipeops.app/.well-known/oauth-authorization-server | jq
curl -i -X POST https://mcp.pipeops.app/mcp
```

The protected-resource document must list `https://mcp.pipeops.app` as its
authorization server. The authorization-server document must advertise PKCE
`S256`, dynamic registration, authorization code, refresh token, and revocation.
The unauthenticated MCP request must return `401` with a `WWW-Authenticate`
challenge containing `resource_metadata`.

Run the repository verification before release:

```bash
go test -race ./...
go vet ./...
go build ./cmd/pipeops-mcp-server
```

Then connect one OAuth client end to end, sign in through Console, request only `api:read`, and verify
that list/get tools appear while create/update/delete/deploy tools do not. Repeat
with `api:write` and verify the full tool surface. Finally, revoke the bridge
OAuth token and end the underlying PipeOps session independently; confirm both
paths stop authorizing requests.

## Future native PipeOps OAuth

The bridge is intentionally replaceable. Once PipeOps exposes a native OAuth
authorization server that supports MCP resource indicators and the required
clients, set `PIPEOPS_OAUTH_MODE=external` and
`PIPEOPS_OAUTH_ISSUER=https://the-native-issuer.example`. No client endpoint
change is required; clients continue using `https://mcp.pipeops.app/mcp`.
