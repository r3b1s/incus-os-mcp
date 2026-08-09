## MODIFIED Requirements

### Requirement: Command execution
- The server SHALL provide a project-aware tool that runs a command in an instance with arguments, working directory, environment, and optional stdin, and captures stdout, stderr, and exit code.
#### Scenario: an-agent-executes-a-command-in-a-running-container-instance
- **WHEN** an agent executes a command in a running container instance in an effective project
- **THEN** the tool returns stdout, stderr, and the exit code; execution uses the Incus exec websocket path for that project.
- **WHEN** an agent executes a command in a VM instance
- **THEN** execution works via the instance's Incus agent when available; otherwise the tool reports that the agent is unavailable.

### Requirement: File push and pull
- The server SHALL provide project-aware tools to write a file into an instance (with mode/uid/gid), read a file from an instance, list a directory, and delete a path.
#### Scenario: an-agent-pushes-a-file-to-an-instance
- **WHEN** an agent pushes a file to an instance with overwrite disabled and the destination exists
- **THEN** the tool fails without replacing the destination and reports that overwrite must be explicitly enabled.
- **WHEN** an agent pushes a file with overwrite enabled
- **THEN** the file is replaced with the requested metadata in the requested effective project.
- **WHEN** an agent pulls a file from an instance
- **THEN** the tool returns the file content (text) or a reference for binary content, plus metadata.
#### Scenario: an-agent-lists-or-deletes-a-path
- **WHEN** an agent lists or deletes a path in an effective project
- **THEN** the operation uses that project scope and honors the recursive flag for directory deletion.
