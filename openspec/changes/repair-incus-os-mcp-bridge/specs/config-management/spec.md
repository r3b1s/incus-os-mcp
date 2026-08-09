## MODIFIED Requirements

### Requirement: Profile management
- The server SHALL provide tools to list, create, update, delete, rename, and copy profiles in the requested effective project.
#### Scenario: an-agent-lists-profiles
- **WHEN** an agent lists profiles with no project override
- **THEN** it receives profile names, config, devices, and used-by references from the configured default project.
- **WHEN** an agent creates, updates, deletes, renames, or copies a profile with a project override
- **THEN** the operation is performed in that project; deletion fails while profiles are in use.
