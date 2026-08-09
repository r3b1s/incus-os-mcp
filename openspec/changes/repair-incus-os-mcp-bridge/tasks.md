## 1. Test foundation and transport contracts

- [x] 1.1 Create an in-process TLS Incus protocol test helper that serves the descriptor/extensions, captures requests, and returns synchronous/asynchronous API responses through the real pinned client.
- [x] 1.2 Add MCP transport tests that invoke registered tools and assert object-shaped structured content plus readable collection content.
- [x] 1.3 Add reusable assertions for effective-project query selection, all-project rejection/selection semantics, operation waiting, and mapped tool errors.

## 2. Project routing and collection-result interoperability

- [x] 2.1 Add a central effective-project/scoped-`InstanceServer` helper and use it for every handler that declares or needs a project input.
- [x] 2.2 Add project inputs to project-scoped resource operations that currently lack one; retain explicitly global server, certificate, project-administration, and storage-pool operations.
- [x] 2.3 Replace top-level slice outputs in all collection handlers with typed `{ "items": [...] }` result envelopes and update registrations/schemas accordingly.
- [ ] 2.4 Cover project routing and collection envelopes across instances, images, storage, networks, profiles, projects, server/certificate operations, and IncusOS application lists.

## 3. Instance and guest file-operation repairs

- [x] 3.1 Route all instance CRUD, lifecycle, state, snapshot, backup, and backup-export calls through the effective project client.
- [x] 3.2 Map instance deletion force/snapshot-cleanup options to target behavior or reject unsupported target behavior; do not silently drop them.
- [x] 3.3 Split instance rename from migration, populate pool/member destinations correctly, and add instance snapshot and backup rename tools.
- [x] 3.4 Route exec and file tools through the effective project client; correct file-push overwrite handling and recursive file-delete behavior.
- [ ] 3.5 Add HTTP-level regression tests for instance deletion, rename/move, snapshot/backup rename, exec/file project scoping, and file mutation flags.

## 4. Image, profile, and storage contract completion

- [x] 4.1 Route image inventory, import, lifecycle, export, and aliases through source effective projects; enforce unambiguous all-project listing.
- [x] 4.2 Pass URL checksums and requested aliases to image import, and route image copy to its requested target project.
- [x] 4.3 Route profile list/create/update/delete/rename/copy operations through their effective project without changing global project administration semantics.
- [x] 4.4 Correct custom-volume creation to send resource type `custom` and separate filesystem/block content type; route volume CRUD/resize through effective projects.
- [x] 4.5 Separate custom-volume rename from migration, then add volume snapshot/backup rename and backup export operations.
- [ ] 4.6 Add project-aware bucket update, bucket-key routing, and HTTP-level regression coverage for image, profile, volume, snapshot, backup, and bucket requests.

## 5. Network, cluster, and IncusOS administration completion

- [x] 5.1 Route managed-network, ACL, zone, and existing zone-record operations through effective projects.
- [x] 5.2 Add project-aware DNS zone-record list, get, and update tools with ETag-safe update behavior.
- [x] 5.3 Add cluster member listing, detail, state inspection, and evacuate/restore action tools required by the server-management contract.
- [x] 5.4 Route IncusOS `/os/1.0` requests through the configured admin client after the existing admin-credential gate.
- [ ] 5.5 Add HTTP-level regression tests for network/project scope, DNS record reads/updates, cluster state/actions, and admin-credential appliance requests.

## 6. Custom ISO-volume import for MicroOS Combustion

- [x] 6.1 Add a project-aware `storage_volume_import_iso` tool that validates a staged regular file and streams it through the official custom-ISO volume import operation.
- [x] 6.2 Register the ISO import tool with clear MCP schemas, operation waiting, and extension/authorization/error reporting.
- [x] 6.3 Add tests for ISO upload endpoint, headers, scoped project query, file validation, and operation results using the TLS protocol helper.
- [x] 6.4 Document the API-only per-run Combustion flow: generate/stage ISO, import custom ISO volume, attach it in the VM-create device map, then clean up the VM and volume.

## 7. Console diagnostics and interactive operator CLI

- [x] 7.1 Add a bounded, project-aware `instance_console_log` MCP tool with explicit truncation reporting and no interactive stream.
- [x] 7.2 Add `incus-os-mcp console <instance>` with common configuration flags, project selection, terminal checks, raw-mode restoration, serial-console data proxying, and resize-control forwarding.
- [ ] 7.3 Promote the existing terminal dependency to direct use if required and add CLI/console session tests without source-text assertions.
- [x] 7.4 Update README, command help, and OpenSpec documentation to distinguish non-interactive MCP diagnostics from interactive CLI console attachment.

## 8. Verification and safe live acceptance

- [x] 8.1 Run `openspec validate repair-incus-os-mcp-bridge --strict`, `go test ./...`, `go vet ./...`, and `go build ./...`; fix all failures at their root cause.
- [x] 8.2 Verify the MCP catalog exposes repaired and new tool schemas, including object-enveloped collection outputs.
- [ ] 8.3 Against the connected IncusOS target, run read-only API validation first, then use explicitly run-unique, disposable ISO-volume and VM resources to validate ISO import, first-boot device attachment, console diagnostics, snapshot/reboot behavior, and cleanup.
- [x] 8.4 Confirm no validation step modified appliance OS/configuration, storage pools, networks, certificate trust, or non-disposable resources; record any blocked live verification precisely.

> 2026-08-08 live acceptance: a run-unique ISO9660-signature probe artifact was imported through `artifact_base64` into a disposable custom ISO volume in project `default`, observed through `storage_volume_list`, attached to a run-unique empty stopped VM at creation with the expected `disk` device (`pool`, `source`, `boot.priority=10`), and verified through `instance_get`. The VM was deleted before its ISO volume, and both were confirmed absent. No pool/network/appliance/certificate/non-disposable resource was changed. Guest first boot, console-log, snapshot, and reboot acceptance remain blocked: this host has no ISO authoring tool or cached real Combustion ISO, and the accessible project has no bootable base image. Do not substitute the probe artifact for a real guest bootstrap test.
