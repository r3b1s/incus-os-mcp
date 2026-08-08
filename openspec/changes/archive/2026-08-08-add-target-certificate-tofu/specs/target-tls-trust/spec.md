## Purpose

Defines how the MCP server establishes, persists, and subsequently enforces trust in an HTTPS target's presented TLS certificate.

## ADDED Requirements

### Requirement: URL-only target configuration
The MCP server SHALL allow an HTTPS target to omit `target.cert_path` and SHALL resolve the omitted value to `target.crt` beside the effective configuration file.

#### Scenario: Default target certificate path
- **WHEN** an HTTPS target configuration supplies a URL but omits `target.cert_path`
- **THEN** the MCP server uses `target.crt` in the effective configuration file's directory as the certificate pin

#### Scenario: Explicit target certificate path
- **WHEN** an HTTPS target configuration supplies `target.cert_path`
- **THEN** the MCP server uses that exact path instead of the conventional path

### Requirement: Trust on first use
When the effective target certificate file is absent, the MCP server SHALL retrieve the leaf certificate presented by the configured HTTPS target, SHALL clearly report that first-use trust occurred with the certificate's SHA-256 fingerprint, and SHALL persist the certificate before making an authenticated Incus API connection.

#### Scenario: First connection to a new target
- **WHEN** the effective target certificate file does not exist and the HTTPS target presents a parseable certificate
- **THEN** the MCP server saves that certificate as the target pin and continues using the saved pin for the authenticated connection

#### Scenario: Certificate acquisition fails
- **WHEN** the target cannot be reached, completes no TLS handshake, or presents no parseable certificate
- **THEN** the MCP server fails without creating a trusted target certificate file

#### Scenario: Concurrent first use
- **WHEN** another process creates the effective target certificate file during first-use acquisition
- **THEN** the MCP server SHALL NOT overwrite that file and SHALL use the certificate that was persisted first

### Requirement: Existing pins fail closed
The MCP server MUST treat an existing target certificate file as authoritative and MUST NOT automatically replace, refresh, or overwrite it.

#### Scenario: Existing pin matches target
- **WHEN** the effective target certificate file exists and the target presents the matching certificate
- **THEN** the MCP server connects using the existing pin without performing first-use enrollment

#### Scenario: Existing pin does not match target
- **WHEN** the effective target certificate file exists and the target presents a different certificate
- **THEN** the MCP server rejects the connection and leaves the existing pin unchanged

#### Scenario: Existing pin is unreadable or invalid
- **WHEN** the effective target certificate file exists but cannot be read or parsed
- **THEN** the MCP server fails without attempting to fetch or replace it

### Requirement: TOFU scope is explicit
The MCP server SHALL apply first-use certificate acquisition only to HTTPS targets and SHALL document that the initial unauthenticated certificate retrieval is vulnerable to interception.

#### Scenario: HTTPS target
- **WHEN** the configured target URL uses HTTPS and no target pin exists
- **THEN** the MCP server performs the documented TOFU workflow

#### Scenario: HTTP target
- **WHEN** the configured target URL uses plain HTTP
- **THEN** the MCP server does not attempt target certificate acquisition
