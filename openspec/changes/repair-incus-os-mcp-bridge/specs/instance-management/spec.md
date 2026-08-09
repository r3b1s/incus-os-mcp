## MODIFIED Requirements

### Requirement: Instance listing and inspection
- The server SHALL expose tools to list instances in the requested effective project (or all projects when explicitly requested), and to fetch a single instance's full config, state, and snapshots from that effective project.
#### Scenario: an-agent-lists-instances
- **WHEN** an agent lists instances with no project override
- **THEN** it receives name, type (container/VM), status, and config for every instance in the configured default project, with the same fidelity as the Incus API.
- **WHEN** an agent supplies a project override or explicitly requests all projects
- **THEN** the request is sent with that project scope or all-projects scope and no unrelated default-project result is substituted.

### Requirement: Instance creation and deletion
- The server SHALL expose instance creation from an image or empty, with config, devices, and profile overrides in the requested effective project; and instance deletion with the requested force and snapshot-cleanup semantics.
#### Scenario: an-agent-creates-an-instance
- **WHEN** an agent creates an instance
- **THEN** the instance appears in the requested effective project with the requested name/config; errors (name conflict, missing image, bad config) surface as tool errors.
- **WHEN** an agent deletes an instance with force or snapshot-cleanup options
- **THEN** the deletion request applies those options and reports the resulting operation; unsupported target behavior is reported as a tool error rather than silently ignored.

### Requirement: Instance rename and move
- The server SHALL expose instance rename and movement to another pool or cluster member in the requested effective project.
#### Scenario: an-agent-renames-or-moves-an-instance
- **WHEN** an agent requests only a new name
- **THEN** the instance is renamed and the tool returns the new name.
- **WHEN** an agent requests movement to another pool or cluster member
- **THEN** the instance migration is requested with the selected destination and the tool returns the completed operation or target error.

### Requirement: Snapshots and backups
- The server SHALL expose project-aware tools to create, list, delete, restore, and rename snapshots; and create, list, delete, rename, and export backups.
#### Scenario: an-agent-creates-a-snapshot-or-backup
- **WHEN** an agent creates a snapshot or backup
- **THEN** the artifact is created and listed among the instance's snapshots/backups in the requested effective project.
- **WHEN** an agent renames a snapshot or backup
- **THEN** the named artifact is renamed in the requested effective project and the resulting operation is reported.
- **WHEN** an agent exports a backup
- **THEN** the tool returns the artifact reference without exposing its contents inline.
