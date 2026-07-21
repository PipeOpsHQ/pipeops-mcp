#!/usr/bin/env bash
# Real-world stdio E2E against api.pipeops.io via pipeops-mcp-server.
# Usage:
#   export PIPEOPS_TOKEN=sat_...
#   ./scripts/e2e_stdio.sh
set -euo pipefail

BIN="${PIPEOPS_MCP_BIN:-$(command -v pipeops-mcp-server || true)}"
if [[ -z "${BIN}" || ! -x "${BIN}" ]]; then
  BIN="$(go env GOPATH)/bin/pipeops-mcp-server"
fi
if [[ ! -x "${BIN}" ]]; then
  echo "pipeops-mcp-server not found; run: go install ./cmd/pipeops-mcp-server" >&2
  exit 1
fi

if [[ -z "${PIPEOPS_TOKEN:-}" ]]; then
  echo "PIPEOPS_TOKEN is required (workspace service token sat_…)" >&2
  exit 1
fi

export PIPEOPS_BASE_URL="${PIPEOPS_BASE_URL:-https://api.pipeops.io}"

python3 - "$BIN" <<'PY'
import json, os, subprocess, sys

bin_path = sys.argv[1]
env = os.environ.copy()

def call(proc, req, expect_response=True):
    proc.stdin.write(json.dumps(req) + "\n")
    proc.stdin.flush()
    if not expect_response:
        return None
    line = proc.stdout.readline()
    if not line:
        raise RuntimeError("no response for " + req.get("method", "?"))
    return json.loads(line)

proc = subprocess.Popen(
    [bin_path],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    env=env,
    text=True,
    bufsize=1,
)
issues = []
try:
    r = call(proc, {
        "jsonrpc": "2.0", "id": 1, "method": "initialize",
        "params": {"protocolVersion": "2024-11-05", "capabilities": {}, "clientInfo": {"name": "e2e", "version": "1.0"}},
    })
    assert "result" in r, r
    print("OK initialize", r["result"].get("serverInfo"))

    call(proc, {"jsonrpc": "2.0", "method": "notifications/initialized"}, expect_response=False)
    print("OK notifications/initialized (no response)")

    r = call(proc, {"jsonrpc": "2.0", "id": 2, "method": "ping", "params": {}})
    assert "result" in r, r
    print("OK ping")

    r = call(proc, {"jsonrpc": "2.0", "id": 3, "method": "tools/list", "params": {}})
    tools = {t["name"] for t in r["result"]["tools"]}
    for need in ("list_workspaces", "list_projects", "create_project", "deploy_project",
                 "get_project", "get_project_build_logs", "list_project_deployments"):
        if need not in tools:
            issues.append(f"missing tool {need}")
        else:
            print("OK tool", need)
    print("OK tools/list count", len(tools))

    def tool(name, arguments=None, rid=10):
        r = call(proc, {
            "jsonrpc": "2.0", "id": rid, "method": "tools/call",
            "params": {"name": name, "arguments": arguments or {}},
        })
        if r.get("error"):
            issues.append(f"{name}: {r['error'].get('message')}")
            print("FAIL", name, r["error"].get("message"))
            return None
        # MCP tool result shape: {content:[{type,text}]}
        content = (r.get("result") or {}).get("content") or []
        text = ""
        for c in content:
            if c.get("type") == "text":
                text += c.get("text") or ""
        print("OK", name, "bytes", len(text))
        return text

    ws_text = tool("list_workspaces", {}, 20)
    workspace_id = None
    if ws_text:
        try:
            data = json.loads(ws_text)
            # flexible shapes
            items = data.get("data") or data.get("workspaces") or data
            if isinstance(items, list) and items:
                workspace_id = items[0].get("UUID") or items[0].get("uuid") or items[0].get("id")
                print("workspace_id", workspace_id)
        except Exception as e:
            issues.append(f"list_workspaces parse: {e}")

    proj_args = {}
    if workspace_id:
        proj_args["workspace_id"] = workspace_id
    proj_text = tool("list_projects", proj_args, 21)
    project_id = None
    if proj_text:
        try:
            data = json.loads(proj_text)
            items = data.get("data") or data.get("projects") or data
            if isinstance(items, dict):
                items = items.get("projects") or items.get("data") or []
            if isinstance(items, list) and items:
                project_id = items[0].get("UUID") or items[0].get("uuid") or items[0].get("id")
                print("project_id", project_id)
        except Exception as e:
            issues.append(f"list_projects parse: {e}")

    if project_id:
        tool("get_project", {"project_id": project_id, **({"workspace_id": workspace_id} if workspace_id else {})}, 22)
        tool("get_project_build_logs", {
            "project_id": project_id,
            **({"workspace_id": workspace_id} if workspace_id else {}),
            "limit": 50,
        }, 23)
        tool("list_project_deployments", {
            "project_id": project_id,
            **({"workspace_id": workspace_id} if workspace_id else {}),
        }, 24)
    else:
        issues.append("no project available to test get_project / build logs")

finally:
    proc.kill()
    err = proc.stderr.read()
    if err.strip():
        print("STDERR:", err[:500])

print("---")
if issues:
    print("ISSUES:")
    for i in issues:
        print(" -", i)
    sys.exit(1)
print("E2E PASSED")
PY
