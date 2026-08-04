## Purpose

Expose Incus profiles and projects as MCP tools: the configuration objects that shape instances and scope tenants.

## ADDED Requirements

### Requirement: Profile management
- The server SHALL tools list, create, update, delete, rename, and copy profiles (per project).
#### Scenario: an-agent-lists-profiles
- **WHEN** an agent lists profiles
- **THEN** it receives profile names, config, devices, and used-by references.
- **WHEN** an agent creates/updates/deletes a profile
- **THEN** the profile is created with the requested config/devices or the error is surfaced; deletion fails while profiles are in use.

### Requirement: Project management
- The server SHALL tools list, create, update, delete, rename, and inspect projects (with resource state).
#### Scenario: an-agent-manages-a-project
- **WHEN** an agent manages a project
- **THEN** projects are created with the requested features/restrictions, and deletion is refused while instances remain unless forced.
