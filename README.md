# incus-os-mcp

An MCP server exposing the IncusOS/Incus REST API (`/1.0` + the IncusOS
appliance API under `/os/1.0`) as typed MCP tools. Built on the official
[Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk)
(v1.7.0+, MCP spec 2026-07-28) and the official
[`lxc/incus` Go client](https://github.com/lxc/incus).

> **Public project.** This documentation carries no personal deployment
> details. All server-specific values (base URL, certificate paths, listen
> address) are configuration, never literals.

## What it is

A single static Go binary with two modes:

- **`run`** — the MCP server: streamable HTTP on `127.0.0.1` (default port
  `8002`), serving typed tools grouped by resource domain.
- **operator CLI** — `config init/show/validate`, `doctor`, `cert setup`.

The tool surface covers instances (CRUD, lifecycle, snapshots, backups,
rename/move), exec + file push/pull, images (import/export/copy/aliases),
storage (pools, volumes, volume snapshots/backups, S3 buckets), networks
(+ ACLs, zones), profiles + projects, and server administration (certificates,
operations, server/cluster state) plus the IncusOS appliance layer (OS updates,
applications, security/recovery keys — admin-credential-gated).

There is **no generic passthrough tool**: every tool is typed with an explicit
schema, so agents get correct parameters and mutations stay explicit.

## Quick start

```sh
# 1. Build
go build -o bin/incus-os-mcp ./cmd/incus-os-mcp

# 2. Write a config file
bin/incus-os-mcp config init   # ~/.config/incus-os-mcp/config.json

# 3. Mint a dedicated client certificate for the server
bin/incus-os-mcp cert setup --dir ~/.config/incus-os-mcp

# 4. Trust it on the target (run on the target host with an admin identity;
#    scope it to a restricted auth group — see docs/bootstrap.md)
incus config trust add ~/.config/incus-os-mcp/mcp-server.crt \
  --type client --restricted --projects default

# 5. Edit the config: target.url, target.cert_path (if self-signed),
#    credential.cert_path, credential.key_path

# 6. Verify the connection
bin/incus-os-mcp doctor

# 7. Run
bin/incus-os-mcp run
```

## Configuration

Precedence: **flags > environment > config file > defaults**.

| Key | Env | Flag | Default | Description |
|---|---|---|---|---|
| `target.url` | `INCUS_MCP_TARGET_URL` | `--target` | `https://127.0.0.1:8443` | IncusOS/Incus base URL |
| `target.cert_path` | `INCUS_MCP_TARGET_CERT` | `--target-cert` | *(empty)* | Target TLS cert (pin for self-signed targets); system CA when empty |
| `credential.cert_path` | `INCUS_MCP_CERT_PATH` | `--cert` | *(required)* | Client TLS certificate (PEM) |
| `credential.key_path` | `INCUS_MCP_KEY_PATH` | `--key` | *(required)* | Client TLS key (PEM) |
| `admin_credential.*` | `INCUS_MCP_ADMIN_*` | `--admin-cert`/`--admin-key` | *(optional)* | Second identity for admin-only surfaces |
| `server.listen_addr` | `INCUS_MCP_LISTEN_ADDR` | `--listen` | `127.0.0.1` | MCP listen address |
| `server.listen_port` | `INCUS_MCP_LISTEN_PORT` | `--port` | `8002` | MCP listen port |
| `default_project` | `INCUS_MCP_PROJECT` | `--project` | `default` | Project when tools omit it |
| `wait_timeout_seconds` | `INCUS_MCP_WAIT_TIMEOUT` | `--wait-timeout` | `60` | Operation wait timeout |
| `inline_max_bytes` | — | — | `1048576` | File-pull inline cap (larger files are staged) |

Example config file:

```json
{
  "target": {
    "url": "https://192.0.2.10:8443",
    "cert_path": "/etc/incus-os-mcp/target.crt"
  },
  "credential": {
    "cert_path": "/etc/incus-os-mcp/client.crt",
    "key_path": "/etc/incus-os-mcp/client.key"
  },
  "server": {
    "listen_addr": "127.0.0.1",
    "listen_port": 8002
  },
  "default_project": "default",
  "wait_timeout_seconds": 60,
  "inline_max_bytes": 1048576
}
```

> Placeholder values are used throughout (documentation-only examples; the
> `192.0.2.0/24` range is reserved for documentation per RFC 5737).

## Authentication model

The MCP server uses a **dedicated client certificate**, minted locally and
trusted on the target — never a human admin certificate, and revocable
independently. Two credential slots:

- **`credential`** (primary): scoped by default — enrolled in a restricted
  auth group on the target (instances/images/storage/network CRUD; read-only
  on server/cluster; no certificate mutation). Permission denials surface as
  clean tool errors, never crashes.
- **`admin_credential`** (optional): a full-access identity for admin-only
  surfaces — certificate management and IncusOS system tools (OS updates,
  applications, recovery keys). Without it, those tools report that admin
  credentials are required.

See [docs/bootstrap.md](docs/bootstrap.md) for enabling fine-grained
authorization on the target and enrolling the server certificate.

## Operation semantics

Mutations wait on their async operation by default (timeout: `wait_timeout_seconds`,
default 60s). On expiry the tool returns the operation ID + status `running`
(not an error); the agent polls with `server_wait_operation`. On failure the
tool errors with the operation ID/URL + error string.

## File push/pull

- **push** fails by default if the destination exists; `overwrite=true`
  replaces. Mode/uid/gid are settable.
- **pull** returns text inline; binary is base64 inline up to
  `inline_max_bytes`; larger files are **staged on the MCP server host** and
  returned as a path reference (large encoded content never enters an agent's
  context window).

## Exec

`instance_exec` runs a command with an argv array (no shell) by default;
`shell=true` opts into `/bin/sh -c`. The result carries `stdout`, `stderr`,
and `exit_code` — a non-zero exit code is a **result field, not a tool error**.
VM exec requires the Incus agent inside the VM; when unavailable the tool
reports it cleanly. Interactive console sessions are **not** part of the MCP
tool surface.

## Deployment

The intended deployment is a containerized user service (e.g. a Podman
quadlet) binding `127.0.0.1`, registered with an MCP client. Loopback-only
binding keeps the transport local; revisit authz if the server is ever
exposed beyond loopback.

## Development

- `go build ./...`, `go vet ./...` — build and vet.
- `cmd/mockincus` — a minimal HTTPS mock of the Incus `/1.0` API for local
  integration testing of the MCP surface without a live target:
  `go run ./cmd/mockincus 127.0.0.1:18443`, then point `--target
  https://127.0.0.1:18443 --target-cert <mock-cert>` at the server.

## License

MIT (see LICENSE).
