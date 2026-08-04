## Purpose

Expose Incus storage management as MCP tools: pools, custom volumes, volume snapshots/backups, and S3 buckets.

## ADDED Requirements

### Requirement: Storage pool management
- The server SHALL tools list, create, update, and delete storage pools (per storage driver and source).
#### Scenario: an-agent-lists-pools
- **WHEN** an agent lists pools
- **THEN** it receives pool names, drivers, sources, and usage.
- **WHEN** an agent creates/updates/deletes a pool
- **THEN** the pool is created with the requested driver/source, or the error is surfaced; deletion fails when volumes are in use.

### Requirement: Custom volume management
- The server SHALL tools list, create, update, delete, rename, move, and resize custom volumes on a pool.
#### Scenario: an-agent-manages-a-custom-volume
- **WHEN** an agent manages a custom volume
- **THEN** the volume is created with the requested config/content type, and mutations are applied per the API.

### Requirement: Volume snapshots and backups
- The server SHALL tools create, list, delete, rename, restore (snapshots), and export (backups) for volumes.
#### Scenario: an-agent-snapshots-or-backs-up-a-volume
- **WHEN** an agent snapshots or backs up a volume
- **THEN** the artifact is created and listed; export returns the artifact reference.

### Requirement: S3 buckets
- The server SHALL tools list, create, update, delete buckets and manage their keys.
#### Scenario: an-agent-manages-a-bucket
- **WHEN** an agent manages a bucket
- **THEN** buckets/keys are created, listed, and deleted per the API.
