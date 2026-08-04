## Purpose

Expose server-level administration as MCP tools: certificate trust, async operation tracking, server/cluster state, and the IncusOS system API (OS updates, applications, security/recovery keys).

## ADDED Requirements

### Requirement: Scoped-certificate operation
The server SHALL operate correctly under a restricted (non-admin) client certificate: Incus permission denials are surfaced as clear tool errors, never crashes; admin-only tools report `forbidden` when the configured cert lacks permission.
#### Scenario: a-scoped-cert-hits-a-permission-denial
- **WHEN** an agent calls a tool the configured certificate is not permitted to perform
- **THEN** the tool returns an explicit permission error identifying the operation, and the server keeps running.
#### Scenario: admin-only-surfaces-with-a-scoped-cert
- **WHEN** the configured cert is scoped and an agent calls a certificate-management or system-management tool
- **THEN** the tool reports that the operation requires admin credentials (per the config's admin-cert setup, if any).

### Requirement: Certificate management
- The server SHALL tools list, add, and delete trusted certificates.
#### Scenario: an-agent-adds-a-certificate
- **WHEN** an agent adds a certificate
- **THEN** the certificate PEM is registered with the requested name/type; deletion removes it from the trust store.

### Requirement: Operation tracking
- The server SHALL tools list operations, fetch a single operation, and wait for completion with a timeout.
#### Scenario: an-agent-calls-a-mutation-tool
- **WHEN** an agent calls a mutation tool
- **THEN** the tool waits for the resulting operation by default and returns its final state; the wait tool allows explicit waiting on any operation URL.
- **WHEN** an operation fails
- **THEN** the tool returns the error description from the operation.

### Requirement: Server and cluster state
- The server SHALL tools report server info (config, API extensions, environment) and cluster state (members, status, evacuation actions).
#### Scenario: an-agent-inspects-server/cluster-state
- **WHEN** an agent inspects server/cluster state
- **THEN** it receives the same information as the API's server and cluster endpoints.

### Requirement: IncusOS system management
- The server SHALL tools manage the IncusOS appliance layer: OS update status/configuration, installed applications, and security information (encryption recovery keys).
#### Scenario: an-agent-queries-OS-update/applications/security-state
- **WHEN** an agent queries OS update/applications/security state
- **THEN** it receives the current status from the IncusOS system API.
- **WHEN** an agent triggers an OS update or application change
- **THEN** the change is applied per the IncusOS system API semantics and its result reported.
- **NOTE**: exact system API routes are pending a spike (initial probe: `/1.0/os/*` not reachable from the standard endpoint); this requirement is finalized from the spike findings before implementation.
