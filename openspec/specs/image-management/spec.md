## Purpose

Expose Incus image management as MCP tools: inventory, import (URL with mandatory sha256, or local file upload), export, copy, delete, refresh, and alias management.

## Requirements

### Requirement: Image inventory and detail
- The server SHALL tools list images (with project/filter) and fetch a single image's metadata.
#### Scenario: an-agent-lists-or-inspects-images
- **WHEN** an agent lists or inspects images
- **THEN** it receives fingerprints, aliases, properties, architecture, and virtual size as reported by the API.

### Requirement: Image import
- The server SHALL tools import images from a URL (sha256 required) and from a local file upload (qcow2/raw for VMs, tarballs for containers).
#### Scenario: an-agent-imports-an-image-from-a-URL-with-a-sha256
- **WHEN** an agent imports an image from a URL with a sha256
- **THEN** the server fetches the image, verifies the checksum, and registers it; verification failure aborts the import with an error.
- **WHEN** an agent imports an image from a local file
- **THEN** the file content is transferred to the Incus server and registered as an image.

### Requirement: Image lifecycle
- The server SHALL tools to delete, copy (to another project/server), export (download artifact), refresh (auto-update), and manage aliases.
#### Scenario: an-agent-deletes,-copies,-exports,-refreshes,-or-aliases-an-
- **WHEN** an agent deletes, copies, exports, refreshes, or aliases an image
- **THEN** the operation is performed per the API, and the tool returns the resulting state (new fingerprint for copy/refresh, artifact reference for export).
