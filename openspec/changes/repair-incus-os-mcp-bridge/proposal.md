## Why

`incus-os-mcp` is intended to be the typed, API-only control plane for IncusOS, but several existing tools ignore declared inputs or call the pinned Incus client incorrectly. In particular, project-scoped requests are sent to the configured default project, custom-volume creation is malformed, and list outputs are incompatible with the connected MCP host. The bridge also lacks operations already promised by its OpenSpec contracts, preventing a reliable disposable openSUSE MicroOS test environment.

## What Changes

- Make every declared project input select the effective Incus project, and preserve explicitly all-project operations where the API supports them.
- Correct existing instance, storage, image, file, and IncusOS-admin tool wiring so declared inputs and documented behavior reach the upstream client/API.
- **BREAKING:** return collection results in an object envelope (for example, `{ "items": [...] }`) rather than top-level array structured content, while retaining readable text content. This makes list tools interoperable with object-only MCP hosts.
- Complete the operations promised by current specs: cluster state/actions, instance and storage snapshot/backup rename and export paths, bucket update, and DNS zone-record read/update operations.
- Add custom-ISO volume import so a per-run Combustion ISO can be streamed into Incus and attached during VM creation without appliance shell access or an Incus CLI.
- Add console support with a bounded read-only MCP console-log tool and an interactive `incus-os-mcp console` CLI command. The MCP surface remains non-interactive; the CLI owns the bidirectional terminal session.
- Add HTTP-level regression coverage for request paths, project query selection, request payloads, collection result shapes, and console/ISO error handling; document the corrected capability boundary.

## Capabilities

### New Capabilities

- `console-access`: Read console logs through MCP and attach an interactive terminal console through the operator CLI.
- `iso-volume-import`: Import a local ISO as an Incus custom ISO storage volume for VM-only, read-only attachment.
- `mcp-result-interoperability`: Provide object-shaped structured content for collection tool responses across MCP hosts.

### Modified Capabilities

- `instance-management`: Honor project selection; correct deletion and rename/move behavior; add the specified snapshot and backup rename operations.
- `instance-exec-files`: Honor project selection and make file overwrite and recursive-delete flags match their declared semantics.
- `storage-management`: Honor project selection; correct custom-volume creation and rename/move behavior; add required snapshot/backup and bucket operations.
- `image-management`: Honor source/target project selection and apply declared checksum and alias behavior during imports/copies.
- `network-management`: Honor project selection and complete DNS zone-record listing, inspection, and update behavior.
- `config-management`: Honor project selection for profile tools while retaining global project administration semantics.
- `server-management`: Complete the specified cluster state/actions and ensure IncusOS appliance requests use the configured admin credential.

## Impact

- Affected code: `internal/server/`, `internal/incus/`, `cmd/incus-os-mcp/`, tests, README, and OpenSpec specifications.
- API impact: collection result structured content changes from arrays to `{ "items": [...] }`; new typed MCP tools and a new interactive CLI subcommand are added.
- Dependencies: uses the already pinned `github.com/lxc/incus/v7` console and ISO-volume APIs; terminal handling may promote the existing `golang.org/x/term` dependency to direct use.
- Infrastructure: validation uses only authenticated IncusOS API/MCP calls against explicitly disposable resources; no appliance shell, OS/configuration change, Proxmox access, or Incus CLI is required.
