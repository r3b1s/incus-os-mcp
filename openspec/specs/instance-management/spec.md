## Purpose

Expose Incus instance management — CRUD, lifecycle control, snapshots, backups, rename/move — as MCP tools so agents can operate instances end to end.

## Requirements

### Requirement: Instance listing and inspection
- The server SHALL the server exposes tools to list instances (with optional project and recursion), and to fetch a single instance's full config, state, and snapshots.
#### Scenario: an-agent-lists-instances
- **WHEN** an agent lists instances
- **THEN** it receives name, type (container/VM), status, and config for every instance in the target project, with the same fidelity as the Incus API.

### Requirement: Instance creation and deletion
- The server SHALL the server exposes instance creation from an image or empty, with config, devices, and profile overrides; and instance deletion with optional snapshot/backup cleanup.
#### Scenario: an-agent-creates-an-instance
- **WHEN** an agent creates an instance
- **THEN** the instance appears in the Incus server with the requested name/config; errors (name conflict, missing image, bad config) surface as tool errors.
- **WHEN** an agent deletes an instance
- **THEN** the instance and its volumes are removed, unless preservation flags are set.

### Requirement: Instance lifecycle control
- The server SHALL tools exist for start, stop, restart, freeze, and unfreeze, each supporting force/wait semantics as the API allows.
#### Scenario: an-agent-starts/stops/restarts/freezes/unfreezes-an-instance
- **WHEN** an agent starts/stops/restarts/freezes/unfreezes an instance
- **THEN** the instance reaches the requested state and the tool reports the final state; asynchronous operations are waited on by default.

### Requirement: Instance rename and move
- The server SHALL renaming and moving (to another pool or cluster member) are exposed.
#### Scenario: an-agent-renames-or-moves-an-instance
- **WHEN** an agent renames or moves an instance
- **THEN** the instance is renamed/moved and the tool returns the new location/name.

### Requirement: Snapshots and backups
- The server SHALL tools to create, list, delete, restore (snapshots), rename, and export (backups) instance snapshots/backups.
#### Scenario: an-agent-creates-a-snapshot-or-backup
- **WHEN** an agent creates a snapshot or backup
- **THEN** the snapshot/backup is created and listed among the instance's snapshots/backups; export returns the backup artifact reference.
