## Context

New repo, no code yet. See proposal.md for motivation. Constraints from exploration (2026-08-03/04):
- Target: any IncusOS/Incus server reachable over HTTPS. Server endpoints are ephemeral and deployment-specific, so base URL, TLS client cert/key, and project defaults are **configuration**, not constants: supplied via config file and/or flags, never hardwired.
- Protocol: `/1.0` REST + websocket operations, TLS client-cert auth.
- Decisions locked with the user: Go + official SDKs; exec + files in phase 1; IncusOS system API in scope (spike required); dedicated server cert; streamable HTTP on 127.0.0.1.
- MCP spec 2026-07-28 (2.0) is supported by both Tier-1 SDKs; stateless core means no session handling.

## Goals / Non-Goals

**Goals:**
- A single static Go binary exposing the full IncusOS surface as typed MCP tools, on the official MCP Go SDK (≥v1.7.0) and official `lxc/incus/v7` client.
- Correct handling of the two hard protocol areas via the official client: exec (websocket) and file push/pull.
- Operations handled uniformly: mutations wait on their operation by default; explicit wait tool.
- Credentials isolated from the human admin cert (dedicated cert, revocable independently).
- The binary ships a CLI for configuring and operating the server itself (config bootstrap/validation, cert paths, run).

**Non-Goals:**
- No interactive instance console/terminal sessions through the MCP tool surface (exec is batch-only).
- No OAuth/MCP 2.0 server authz yet (loopback-only transport; revisit if exposed beyond 127.0.0.1).
- No OpenFGA scoping of the server cert (full-admin cert, documented).
- No web UI.

## Decisions

### D1: Go, official SDKs
`github.com/modelcontextprotocol/go-sdk` v1.7.0+ (MCP 2.0, 2026-07-28) + `github.com/lxc/incus/v7/client` (all websocket/file/exec protocol logic done, typed, upstream-tested). Alternatives: Python (re-implements exec/files websockets by hand; rejected), hand-rolled REST (rejected — duplicates the official client). Single static binary = smallest attack surface for a full-admin cert.

### D1a: Everything server-specific is configuration
Base URL, TLS client cert/key paths, listen address/port, and default project come from config file + flags (environment variables for secrets). No endpoint, hostname, or IP is compiled in or documented as a literal; README documents the config schema with placeholder examples.

### D1b: The binary has a CLI
Same binary, two modes: `run` (the MCP server) and config subcommands (`config init`/`config show`/`config validate`, cert path setup, `doctor` for connection checks). One flagset, one config loader — no separate tooling, useful for bootstrapping and debugging the server itself.

### D2: Tool architecture — resource-grouped tools over a thin API client wrapper
One internal `incus.Client` wrapper (config: URL, cert/key paths) used by all tools. Tools grouped per capability; each tool is a thin typed function: request → (wait op) → structured result. No generic passthrough tool: typed tools give agents correct schemas and keep mutations explicit. Tool count target ≈ 40–50.

### D3: Operation semantics
Mutations return operations from the API. Default: wait for completion (timeout + error surfacing) and return final state. A `operation_wait` tool accepts any operation URL/ID for manual waiting. Rationale: agents reason about final states, not async handles.

### D4: Certificates
Server uses its own client cert (paths from config; e.g. a dedicated config directory) minted with `openssl` and trusted via `POST /1.0/certificates` using an existing admin cert. The MCP server never holds a human admin cert.

### D5: Transport and deployment
Streamable HTTP on 127.0.0.1 (listen address/port configurable; suggested default 8002) via the SDK's HTTP transport; containerized user service (e.g. podman quadlet); registered with an MCP client. Stateless core (2.0) = no session affinity needed.

### D6: IncusOS system API — spike first
The `incus admin os` surface (updates, applications, security/recovery keys) needs route discovery: attempt `/1.0/os/*`, `/os/*`, and the osd's own listener via the Go client's custom endpoint support. Implementation of system tools is gated on spike findings; spec requirement `server-management` is finalized from them.

### D7: Project support
All instance/image/volume tools accept an optional `project` parameter (default `default`), surfaced as tool schema fields.

## Risks / Trade-offs

- **Full-admin cert**: the server inherits complete control of IncusOS. Mitigation: loopback-only transport, read-mostly design pressure via typed tools, dedicated revocable cert. OpenFGA scoping is a possible follow-up.
- **VM exec dependency**: exec on VMs requires the Incus agent inside the VM; tools must report "agent unavailable" cleanly (spec requirement).
- **System API unknown surface**: spike may reveal osd routes are unix-socket-only on the appliance — in that case system tools degrade to what's reachable over the network and the shortfall is reported, not hacked around.
- **MCP 2.0 freshness**: go-sdk 1.7.0 is the first 2.0 release line; pin the version, upgrade deliberately.
- Binary file pull: returned as artifact/reference rather than inline content where size makes inline impractical.
