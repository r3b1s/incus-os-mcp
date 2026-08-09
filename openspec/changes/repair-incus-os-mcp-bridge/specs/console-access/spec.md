## Purpose

Provide diagnosable VM boot access through bounded MCP console logs and a secure interactive serial-console command in the bridge operator CLI.

## ADDED Requirements

### Requirement: Read-only console logs through MCP
The server SHALL provide a project-aware MCP tool that reads an instance console log without opening an interactive console session. The tool SHALL return bounded log content and indicate when configured output limits truncate the available log.

#### Scenario: an-agent-inspects-a-failed-vm-boot
- **WHEN** an agent requests the console log for a VM in an effective project
- **THEN** it receives the available serial-console log content or a clear target error without requiring an instance agent

#### Scenario: a-console-log-exceeds-the-inline-limit
- **WHEN** an instance console log exceeds the tool's configured response limit
- **THEN** the result identifies that the returned content was truncated rather than presenting it as the complete log

### Requirement: Interactive serial console through the operator CLI
The `incus-os-mcp` executable SHALL provide a `console` subcommand that attaches the caller's terminal bidirectionally to an instance serial console in the requested effective project. It SHALL restore the local terminal when the session exits and surface unsupported-console, authorization, and connection failures as command errors.

#### Scenario: an-operator-attaches-to-a-vm-console
- **WHEN** an operator invokes `incus-os-mcp console <instance>` from a terminal with valid target credentials
- **THEN** the operator can exchange terminal input and output with the VM serial console until the session is detached or exits

#### Scenario: an-agent-needs-an-interactive-shell
- **WHEN** an agent needs an interactive shell
- **THEN** the MCP tool surface remains non-interactive and the operator CLI provides the explicit console attachment path
