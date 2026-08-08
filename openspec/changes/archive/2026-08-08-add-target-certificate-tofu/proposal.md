## Why

A newly provisioned IncusOS target presents a self-signed TLS certificate, but operators currently must retrieve that certificate manually before `incus-os-mcp` can connect securely. The MCP server should support an explicit trust-on-first-use bootstrap so the target configuration only needs its URL while preserving certificate pinning after the first connection.

## What Changes

- When an HTTPS target certificate path is not configured, use the conventional `target.crt` path in the MCP server's configuration directory.
- If the effective target certificate file does not exist, retrieve the leaf certificate presented by the configured target, report its SHA-256 fingerprint, and persist it atomically as the target pin.
- Reuse an existing target certificate without network re-enrollment and never overwrite or rotate it automatically.
- Fail closed when the pinned certificate no longer matches; operators must deliberately remove or replace the pin to trust a reinstalled or changed target.
- Document the accepted TOFU boundary and URL-only target configuration.

## Capabilities

### New Capabilities

- `target-tls-trust`: First-use target certificate acquisition, durable pinning, and fail-closed reuse semantics.

### Modified Capabilities

<!-- None. The original change has not been archived into baseline specs. -->

## Impact

- Affects target configuration resolution and Incus client connection setup.
- Adds filesystem persistence under the MCP server's existing configuration directory.
- Adds tests for first-use acquisition, pin reuse, explicit paths, and mismatch handling.
- Changes empty `target.cert_path` behavior for HTTPS targets from system-CA verification to explicit TOFU pinning.
