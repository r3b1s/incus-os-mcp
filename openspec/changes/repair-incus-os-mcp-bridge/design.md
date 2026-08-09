## Context

See `proposal.md` for motivation and the completed delta specs for behavior. The bridge uses a single official Incus Go client wrapper shared by typed resource tools, but many handlers call the unscoped connection directly. The configured default project is therefore only displayed, not applied to normal operations. Several declared options are similarly disconnected from upstream request fields.

The current MCP host rejects array-valued `structuredContent`, even though the pinned SDK accepts it. The bridge also has no server-layer tests that exercise actual Incus request paths or generated MCP response shapes. Finally, the current OpenSpec contracts promise several resource operations that are not registered, and the operator binary has no instance-console command.

The design must preserve the appliance boundary: IncusOS is an authenticated hypervisor API only. The workflow must not depend on an appliance shell, an Incus CLI, Proxmox, guest-agent execution, or changes to appliance OS, networking, storage, certificates, or project configuration outside explicitly disposable test resources.

## Goals / Non-Goals

**Goals:**

- Make every documented project-bearing tool address its effective project consistently.
- Repair existing tool-to-client request construction and complete the operations required by current OpenSpec contracts.
- Make collection results portable to object-only MCP hosts without adding generic passthrough tools.
- Support API-only upload and VM attachment of a per-run Combustion ISO.
- Provide both automated console diagnostics and an operator-owned interactive serial console.
- Add behavior-level tests that issue real HTTP requests through the pinned Incus client against an in-process TLS mock.

**Non-Goals:**

- Mirroring every current or future Incus REST endpoint, cluster-group endpoint, or undocumented IncusOS endpoint.
- Adding an interactive terminal session to the MCP tool surface or forwarding console byte streams through chat.
- Changing the IncusOS appliance, its default project, storage pools, networks, certificates, or security configuration.
- Replacing the official Incus client with a generic raw REST passthrough.
- Supporting direct client-to-appliance ISO file paths; ISO source artifacts remain files staged on the MCP server host.

## Decisions

### 1. Resolve a scoped official client once per request

Handlers with a `project` input will obtain an `InstanceServer` scoped with the effective project (explicit input or configured default) through a central wrapper helper. All subsequent Incus calls in that handler use this scoped client. Inputs that presently omit `project` despite resource-level project semantics will gain it.

Explicit all-project list operations will use the root connection's all-project methods and reject a simultaneous project override as ambiguous. Global resources—server information, certificate trust, project administration, and storage-pool administration—remain unscoped where Incus defines them as global.

**Rationale:** `UseProject` preserves the official client's operation, TLS, extension, and error behavior while preventing a repeat of the current “declared but ignored” project option across resource groups.

**Alternatives considered:**

- Append `project` query parameters manually in each handler. Rejected: duplicates URL behavior and misses websocket/download paths.
- Set one process-global active project. Rejected: the streamable MCP server serves concurrent requests and project scope must be per tool call.

### 2. Model collection responses as typed object outputs

Collection handlers will return a common generic object such as `ListOutput[T]{Items []T}`. The output type is changed at the handler boundary so the MCP SDK emits an object schema and object-shaped `structuredContent`; the readable text result remains available through the SDK's normal serialization.

**Rationale:** changing only the generic result helper would leave generated tool schemas and typed output values array-shaped. An explicit envelope gives callers a stable, uniform collection contract.

**Alternatives considered:**

- Retain raw arrays because the MCP SDK permits them. Rejected: the connected host rejects them before the agent can use discovery tools.
- Add host-specific conditionals. Rejected: tool responses should be portable and deterministic.

### 3. Use the pinned Incus client for corrected and missing operations

Existing mutations will call the specific client operation that matches the requested intent: rename APIs for renames, migration APIs only with migration requests, and `custom` resource type plus separate volume content type for custom-volume create. Handler options are either mapped to an upstream request/query field or removed/rejected; none will be silently ignored.

The bridge will add the missing OpenSpec operations in their existing resource groups: instance snapshot/backup rename, volume snapshot/backup rename and export, bucket update, network-zone record list/get/update, and cluster membership/state/evacuation operations. IncusOS raw appliance requests will use the configured admin client after the existing admin-credential gate succeeds.

**Rationale:** resource-specific typed methods retain extension checks and established operation waiting. A narrowly internal request helper is allowed only where the official interface lacks an option required by the bridge's existing contract; it will not be exposed as a generic MCP API tool.

### 4. Import per-run ISO artifacts through custom ISO volumes

`storage_volume_import_iso` accepts pool, name, project, a standard-base64 ISO artifact, and wait timeout. It limits decoded input to 64 MiB, decodes the artifact to an ephemeral temporary file, verifies the ISO9660 primary-volume descriptor, streams that file through the official custom-ISO upload operation on the scoped client, and deletes it on every return path. The tool returns the completed operation and exposes neither appliance nor MCP-server filesystem paths.

The existing VM-create devices map is the attachment mechanism. Test orchestration will generate a run-unique ISO, import it, include a read-only disk device that references the named custom ISO volume during VM creation, and delete both resources during cleanup.

**Rationale:** this follows the upstream `custom_volume_iso` implementation while keeping data movement API-only from both the appliance and the MCP caller's perspective. The explicit input and decoded-byte limits bound request/context and temporary-disk consumption.

**Alternatives considered:**

- Host-path disk devices. Rejected: they require a file on the IncusOS appliance.
- Cloud-init configuration media. Rejected: it is not a Combustion volume and cannot substitute for MicroOS first-boot configuration.
- A staged MCP-server file path. Rejected: a remote MCP bridge cannot assume its caller can stage host-local files safely, and it leaks a deployment-specific path into the tool contract.

### 5. Split read-only MCP diagnostics from interactive CLI console attachment

`instance_console_log` will be a project-aware, bounded read-only MCP tool. It will read console output through the official client without relying on an Incus guest agent and expose whether the configured byte cap truncated output.

`incus-os-mcp console <instance>` will be an operator CLI command using the existing configuration/credential flags. It will require terminal stdin/stdout, switch the terminal to raw mode for the session, proxy the serial-console data channel, propagate window-size changes through the console control channel, and restore terminal state on every exit path. The command will use the effective project through the same scoping helper.

**Rationale:** MCP request/response semantics cannot safely host an unbounded bidirectional terminal stream, while the CLI already has the process, credentials, signal handling, and terminal ownership required by the upstream console API.

**Alternatives considered:**

- An interactive `instance_console` MCP tool. Rejected: it would claim a persistent stream in a request/response tool protocol and conflict with the existing non-interactive MCP boundary.
- Console log only. Rejected: it cannot support real-time recovery and manual MicroOS bootstrap diagnostics.

### 6. Verify behavior with an in-process HTTPS Incus protocol mock

Tests will use a configurable `httptest` TLS server and the real pinned Incus client. The mock will provide the server descriptor/extensions and capture request method, path, query, headers, and payload. MCP handler tests will call the actual server transport to verify generated response schemas and structured outputs; CLI console internals will be factored around a small terminal/session boundary so raw terminal restoration and control messages can be tested without source-text assertions.

**Rationale:** interface mocks hide URL, scoping, operation, and serialization defects—the defect class found in this audit.

## Risks / Trade-offs

- **Breaking collection structured-content shape** → Document `{ "items": [...] }`, retain readable content, and include explicit migration notes before deployment.
- **Project routing touches many handlers** → Centralize scoped-client selection and add request-query regression coverage for every resource group with project inputs.
- **Some API options are not represented by a convenient pinned client method** → Use a narrowly typed internal helper only after confirming the official REST contract; reject unsupported options instead of silently discarding them.
- **Console sessions can leave a local terminal unusable on error** → Defer terminal restoration immediately after raw-mode entry, handle signals, and unit-test all error paths.
- **Console logs or ISO artifacts can be large** → Bound returned log bytes, reject ISO artifacts over 64 MiB decoded, and use an ephemeral file only for the client's streaming upload API.
- **Live appliance testing can create durable resources** → Use run-unique names, a recorded cleanup sequence, and only disposable volumes/VMs after unit and mock validation succeeds.

## Migration Plan

1. Release the bridge with the documented collection envelope and new tools/CLI command.
2. Update MCP consumers to read collection data from `items`; no server-side data migration is needed.
3. Restart the bridge service so the new tool catalog and external OpenSpec skills are loaded.
4. Run local strict OpenSpec validation, Go tests, vet, and build before a live target check.
5. Run live API-only validation using read-only discovery first, then a run-unique ISO volume and disposable VM; remove the VM and ISO volume even on a failed bootstrap.
6. Roll back by deploying the prior bridge binary if a client cannot yet consume object-enveloped lists. No IncusOS appliance rollback is required.
