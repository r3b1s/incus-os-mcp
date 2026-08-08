# incus-os-mcp

An MCP server exposing the Incus/IncusOS REST API as typed MCP tools. Public project; docs carry no personal deployment details.

## Language

**MCP server**:
The binary this repo builds — an MCP tool provider. Never bare "server" in docs/config.
_Avoid_: server

**Config precedence**:
flags > environment > config file > defaults. Flags are per-invocation overrides; env carries secrets (containerized deploys); file sets baselines.

**Logging**:
Structured JSON lines to stdout/stderr, `info` default, `debug` for tool-call traces. Log tool name, operation ID, duration, outcome. Never credentials, keys, tokens, or target URLs beyond host:port.

**Events & metrics** (non-goal):
Incus events websocket and Prometheus metrics are out of the tool surface. Metrics are consumed by Prometheus directly with its own credential; events don't fit the request/response tool model (revisit only via an MCP extension, not a tool).

**Target**:
The IncusOS/Incus host the MCP server talks to (its base URL + TLS credential are config). `target_url`, `target_cert` in config.
_Avoid_: server, remote, host

**Target trust**:
The HTTPS target's presented leaf certificate pinned as PEM. An explicit `target.cert_path` wins; otherwise the pin is `target.crt` beside the effective config file. If absent, `run`/`doctor` acquire it by visible trust on first use (path + SHA-256 fingerprint), persist before any Incus API request, and never overwrite it automatically. A mismatch fails closed until the operator deliberately replaces or removes the pin.

**Incus server**:
Incus's own server-side behavior, when discussed as such (e.g. "the Incus server enforces permissions").

**Instance**:
Any runnable unit on the target; subtypes are **container** and **VM**, sharing one API. Tool prefixes use `instance_*`; subtype matters only where behavior differs (e.g. exec needs the agent on VMs).
_Avoid_: LXC container, LXD instance

**Container** / **VM**:
The two instance subtypes, named exactly as Incus names them.

**Certificate**:
The TLS artifact (PEM, fingerprint, the trust-store entry at `/1.0/certificates`).

**Credential**:
The configured identity the MCP server uses: a cert+key pair with an access level — **scoped credential** (restricted auth group) or **admin credential** (full access). Config: `credential.*` and optional `admin_credential.*`.
_Avoid_: client cert (only when Incus's own trust-store view is meant)

**Project**:
The Incus tenant namespace (instances/images/volumes scoped per project; default `"default"`). Tool param + config, lowercase. Never used for the repo itself. Tools fall back to the configured `default_project` (default `"default"`) when the param is omitted.
_Avoid_: repo-as-project

**Incus API**:
The standard `/1.0` surface of the target (instances, images, storage, networks, …).

**Appliance API**:
The IncusOS-specific admin surface (OS updates, applications, recovery keys; routes pending spike). Tool prefix `appliance_*`.
_Avoid_: system API, osd

**Recovery keys**:
The target's LUKS break-glass keys. Retrieval is an `appliance_*` tool gated to the **admin credential** only — a scoped credential gets an explicit "requires admin credential" error. Key material is never logged.

**Doctor**:
The CLI's connectivity/health check: (1) config parse + precedence, (2) cert/key load + file perms, (3) TLS handshake + `/1.0` reachable (version + API extensions), (4) effective-permissions probe (read-only list; admin check if admin creds configured). Non-zero exit on any failure.

**Operation**:
The Incus async task object (URL/ID/status/error). MCP tools wait and return final state; `operation_wait` accepts an operation URL/ID. Generic actions are "tool calls".
_Avoid_: operation-as-any-action

**Wait semantics**:
Default wait timeout 60s (configurable). On expiry, return the operation ID + status `running` as a normal result (not an error); the agent polls with `operation_wait`. Long operations document their expected duration in tool descriptions. On operation failure, the tool errors with the operation ID/URL + error string.

**Capability**:
The openspec spec unit (`instance-management`, `server-management`, …). MCP-side feature negotiation is "MCP features" / "MCP extensions" (2.0 `ext-*`).
_Avoid_: capability-for-mcp-negotiation

**Exec**:
Batch command execution on an instance. Result carries stdout, stderr, and `exit_code`; non-zero exit codes are result fields, never tool errors. Tool errors are transport/auth/protocol failures only. Input is an argv array (no shell) by default; a `shell` flag (default off) opts into `/bin/sh -c`.

**File pull**:
Fetching a file from an instance via the MCP server (the credential holder). Text inline; binary ≤ cap (default 1 MiB, configurable) as base64 inline; larger files return a **staged-file reference** (path + size on the MCP server host). Rule: encoded content representing large files must never enter an agent's context window.
_Avoid_: pulling large files inline

**File push**:
Writing a file into an instance via the MCP server. Fails by default if the destination exists; explicit `overwrite=true` allows replacement. Mode/uid/gid settable. Delete is explicit (recursive flag for dirs).

**Image import**:
Registering an image on the target from a URL (mandatory sha256) or file upload. The MCP server pre-validates sha256 form (64 hex) and URL scheme before the API call; Incus's own verification remains the authoritative gate.
