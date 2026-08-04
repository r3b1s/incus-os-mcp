## 1. Spike: IncusOS system API surface

- [ ] 1.1 Map the `incus admin os` surface (updates, applications, security/recovery keys): try `/1.0/os/*`, `/os/*`, custom-endpoint support in the Go client, and the osd listener; record exact routes + auth
- [ ] 1.2 Finalize `server-management` spec requirement from spike findings (adjust spec if routes differ from assumptions)

## 2. Project scaffold

- [ ] 2.1 `go mod init` (module per repo convention); pin `github.com/modelcontextprotocol/go-sdk` ≥v1.7.0 and `github.com/lxc/incus/v7`
- [ ] 2.2 Config: base URL, TLS cert/key paths (optional separate admin cert), listen address/port, default project — config file + flags (env for secrets), placeholder docs; internal `incus` client wrapper using it
- [ ] 2.3 MCP server skeleton on go-sdk (streamable HTTP transport, 127.0.0.1, configurable port) with one probe tool (`server_info`)
- [ ] 2.4 Error mapping: Incus 403/permission errors → explicit tool errors (scoped-cert graceful degradation); bootstrap docs for `incus auth` + restricted auth group on the target

## 3. Core tools: instances + operations

- [ ] 3.1 Instance tools: list/get/create/delete, lifecycle (start/stop/restart/freeze/unfreeze), rename/move, state
- [ ] 3.2 Snapshot tools: create/list/delete/restore/rename; backup tools: create/list/delete/export
- [ ] 3.3 Operation helper: wait-on-operation default for all mutations + `operation_wait` tool (list/get/wait with timeout)

## 4. Exec and files

- [ ] 4.1 `instance_exec` tool (container native, VM-agent path, stdout/stderr/exit code; agent-unavailable error path)
- [ ] 4.2 File tools: push (mode/uid/gid), pull (text inline, binary as reference), list directory, delete (recursive flag)

## 5. Remaining resource groups

- [ ] 5.1 Image tools: list/get/import (URL+sha256, file upload)/delete/copy/export/refresh + aliases
- [ ] 5.2 Storage tools: pools CRUD, volumes CRUD + resize/rename/move, volume snapshots/backups, buckets + keys
- [ ] 5.3 Network tools: networks CRUD, ACLs, zones
- [ ] 5.4 Config tools: profiles CRUD/copy/rename, projects CRUD/rename + state
- [ ] 5.5 Server tools: certificates list/add/delete, server info, cluster state; system tools per spike findings (updates, applications, security/recovery keys)

## 6. Verification

- [ ] 6.1 Integration pass against a live IncusOS instance (configurable endpoint): smoke-test each tool group (read + one mutation each) with the dedicated cert
- [ ] 6.2 Security review: loopback-only binding, cert file permissions, no secrets in tool output, error paths don't leak creds
- [ ] 6.3 Containerized deployment + MCP client registration; end-to-end agent call against the deployed server
