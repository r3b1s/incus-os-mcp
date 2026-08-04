## Purpose

Expose command execution and file transfer on Incus instances as MCP tools — the container-native exec and instance file push/pull capabilities.

## ADDED Requirements

### Requirement: Command execution
- The server SHALL a tool runs a command in an instance, with arguments, working directory, environment, and optional stdin; captures stdout/stderr and exit code.
#### Scenario: an-agent-executes-a-command-in-a-running-container-instance
- **WHEN** an agent executes a command in a running container instance
- **THEN** the tool returns stdout, stderr, and the exit code; execution uses the Incus exec websocket path.
- **WHEN** an agent executes a command in a VM instance
- **THEN** execution works via the instance's Incus agent when available; otherwise the tool reports that the agent is unavailable.

### Requirement: File push and pull
- The server SHALL tools to write a file into an instance (with mode/uid/gid), read a file from an instance, list a directory, and delete a path.
#### Scenario: an-agent-pushes-a-file-to-an-instance
- **WHEN** an agent pushes a file to an instance
- **THEN** the file is written with the requested metadata; overwrite semantics are explicit.
- **WHEN** an agent pulls a file from an instance
- **THEN** the tool returns the file content (text) or a reference for binary content, plus metadata.
#### Scenario: an-agent-lists-or-deletes-a-path
- **WHEN** an agent lists or deletes a path
- **THEN** the operation succeeds for files and directories per the API semantics (recursive flag for directories).

### Requirement: Interactive sessions excluded from the MCP tool surface
The server SHALL NOT expose interactive terminal (console) sessions on instances through the MCP tool surface; exec is non-interactive batch execution only. The server binary's own CLI is unaffected.
#### Scenario: an-agent-needs-an-interactive-shell
- **WHEN** an agent needs an interactive shell
- **THEN** the MCP tool surface does not provide one; the API console remains available out-of-band.
