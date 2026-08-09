## MODIFIED Requirements

### Requirement: Image inventory and detail
- The server SHALL provide tools to list images with requested effective-project, all-projects, and filter scope, and to fetch a single image's metadata from its effective project.
#### Scenario: an-agent-lists-or-inspects-images
- **WHEN** an agent lists or inspects images with a project override
- **THEN** it receives fingerprints, aliases, properties, architecture, and virtual size from that project as reported by the API.
- **WHEN** an agent explicitly requests all projects
- **THEN** the tool uses the API's all-projects scope rather than replacing it with the configured default project.

### Requirement: Image import
- The server SHALL import images into the requested effective project from a URL with a mandatory SHA-256 checksum or from a local file upload, and SHALL apply requested aliases.
#### Scenario: an-agent-imports-an-image-from-a-URL-with-a-sha256
- **WHEN** an agent imports an image from a URL with a SHA-256 checksum
- **THEN** the server passes the checksum to Incus for verification and registers the image in the requested project; verification failure aborts the import with an error.
- **WHEN** an agent imports an image from a local file with aliases
- **THEN** the file content is transferred to the Incus server, registered in the requested project, and the requested aliases are created.

### Requirement: Image lifecycle
- The server SHALL provide project-aware tools to delete, copy to another project or server, export, refresh, and manage aliases.
#### Scenario: an-agent-deletes,-copies,-exports,-refreshes,-or-aliases-an-
- **WHEN** an agent deletes, copies, exports, refreshes, or aliases an image
- **THEN** the operation is performed with the requested source and target project scope, and the tool returns the resulting state (new fingerprint for copy/refresh, artifact reference for export).
