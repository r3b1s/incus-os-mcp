## MODIFIED Requirements

### Requirement: Custom volume management
- The server SHALL provide project-aware tools to list, create, update, delete, rename, move, and resize custom volumes on a pool. Filesystem or block content type SHALL be sent as volume content type while the resource type remains custom.
#### Scenario: an-agent-manages-a-custom-volume
- **WHEN** an agent creates a custom volume with requested config and content type in an effective project
- **THEN** the volume is created as a custom volume with that content type and configuration.
- **WHEN** an agent renames or moves a custom volume
- **THEN** a rename is performed as a rename and a move is performed as a migration to the requested destination; unsupported target behavior is reported rather than silently ignored.

### Requirement: Volume snapshots and backups
- The server SHALL provide project-aware tools to create, list, delete, rename, and restore volume snapshots; and create, list, delete, rename, and export volume backups.
#### Scenario: an-agent-snapshots-or-backs-up-a-volume
- **WHEN** an agent snapshots or backs up a volume in an effective project
- **THEN** the artifact is created and listed in that project.
- **WHEN** an agent renames a volume snapshot or backup
- **THEN** the named artifact is renamed and the resulting operation is reported.
- **WHEN** an agent exports a volume backup
- **THEN** the tool returns an artifact reference without exposing backup contents inline.

### Requirement: S3 buckets
- The server SHALL provide project-aware tools to list, create, update, and delete buckets and to manage their keys.
#### Scenario: an-agent-manages-a-bucket
- **WHEN** an agent creates, updates, lists, or deletes a bucket in an effective project
- **THEN** the bucket configuration and description are applied or returned for that project.
- **WHEN** an agent manages a bucket key
- **THEN** the key operation is applied to the selected bucket in that effective project.
