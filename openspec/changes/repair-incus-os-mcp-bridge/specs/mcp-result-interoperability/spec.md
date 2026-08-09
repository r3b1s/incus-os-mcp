## Purpose

Make every collection result from the MCP bridge portable across hosts that require structured content to be a JSON object rather than a top-level array.

## ADDED Requirements

### Requirement: Collection results use an object envelope
The server SHALL return successful collection results as object-shaped structured content with an `items` array containing the same API resources that the equivalent Incus collection endpoint returns. This requirement applies to every MCP tool whose primary successful result is a collection, including list operations for instances, images, storage resources, networks, profiles, projects, certificates, operations, and IncusOS applications.

#### Scenario: an-agent-lists-a-resource-collection
- **WHEN** an agent calls a collection tool against a target that returns zero or more resources
- **THEN** the tool result structured content is a JSON object with an `items` array and no top-level array structured content is emitted

#### Scenario: a-host-requires-object-structured-content
- **WHEN** an object-only MCP host invokes a collection tool
- **THEN** it receives the collection without a structured-content validation error
