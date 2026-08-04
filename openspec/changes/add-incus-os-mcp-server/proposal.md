## Why

IncusOS provides a rich REST API (instances, exec, files, images, storage, system admin) but no MCP tooling exists for it: the only community server wraps a local `lxc` CLI (inapplicable — IncusOS has no shell, remote REST only), and the API surface is currently reachable only via hand-rolled curl. Agents should be able to operate IncusOS through typed MCP tools.

## What Changes

- New Go MCP server (`incus-os-mcp`) exposing the IncusOS REST API (`/1.0` + IncusOS system API) as typed MCP tools, built on the official MCP Go SDK (v1.7.0+, MCP spec 2026-07-28) and the official `lxc/incus/v7` Go client.
- Tool surface grouped by resource domain: instances (CRUD, lifecycle, snapshots, backups, rename/move), exec + file push/pull, images (import incl. URL+sha256 and file upload, export, copy, aliases), storage (pools, volumes, volume snapshots/backups, buckets), networks (+ ACLs, zones), profiles + projects, server admin (certificates, operations, server/cluster state, IncusOS system: updates, applications, security/recovery keys).
- Authentication: dedicated TLS client certificate (minted locally, trusted via `POST /1.0/certificates` using the existing admin cert), scoped to the MCP server and revocable independently.
- Transport: streamable HTTP on 127.0.0.1 (shared server; other clients can attach).
- Async Incus operations surfaced via operation-aware tools (wait-on-operation by default, with a standalone `wait` tool).
- **BREAKING**: none for existing systems; new server, new port.

## Capabilities

### New Capabilities
- `instance-management`: instances — list/get/create/delete, lifecycle (start/stop/restart/freeze/unfreeze), rename/move, state, snapshots, backups
- `instance-exec-files`: command execution (container native, VM via agent) and file push/pull/delete on instances
- `image-management`: images — list/get/import (URL with sha256, file upload)/delete/copy/export/refresh + aliases
- `storage-management`: storage pools, volumes, volume snapshots/backups, buckets (CRUD)
- `network-management`: networks, ACLs, zones (CRUD)
- `config-management`: profiles and projects (CRUD)
- `server-management`: certificates (list/add/delete), operations (list/get/wait), server info, cluster state, IncusOS system API (OS updates, applications, security/recovery keys)

### Modified Capabilities
<!-- none — no existing specs in this repo -->

## Impact

- New repository component: `cmd/incus-os-mcp` (binary) + internal packages; uv-free, single static Go binary.
- Dependencies: `github.com/modelcontextprotocol/go-sdk` (≥v1.7.0), `github.com/lxc/incus/v7` client.
- New credentials: dedicated IncusOS client cert (in the server's config dir), trusted on the target server.
- Deployment: containerized user service (e.g. podman quadlet), streamable HTTP on 127.0.0.1 (port configurable), registered with an MCP client.
- Requires a spike: exact routes of the IncusOS system API (osd) — `/1.0/os/*` returned 404 from outside; surface must be mapped before system tools can be specified fully.
