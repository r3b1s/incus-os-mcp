## Context

See `proposal.md` for motivation. Today `target.cert_path` is optional, and an empty value delegates HTTPS verification to the system CA pool. IncusOS normally presents a self-signed certificate, so operators manually retrieve and configure a PEM file before the first connection. Configuration already has a deterministic effective file path and the Incus client wrapper centralizes every target connection.

## Goals / Non-Goals

**Goals:**

- Make the target subsection URL-only for the common IncusOS case.
- Restrict insecure verification to one certificate-retrieval handshake.
- Turn the retrieved certificate into a durable, fail-closed pin before any Incus API request.
- Make first-use trust visible through path and SHA-256 fingerprint output.

**Non-Goals:**

- Automatically trusting a changed or renewed certificate.
- Certificate rotation, ACME, or PKI management.
- Removing the requirement for a configured client credential.
- Applying TLS semantics to plain HTTP targets.

## Decisions

### D1: Resolve an omitted path beside the effective config file

The effective pin path is an explicit `target.cert_path` when set; otherwise it is `target.crt` in the directory containing the loaded config file. This respects custom `--config` and `INCUS_MCP_CONFIG` locations and keeps the config portable. The resolved runtime path does not need to be serialized into JSON.

Alternative: always use the XDG default path. Rejected because containerized and custom-config deployments would unexpectedly write outside their mounted configuration directory.

### D2: Fetch the presented leaf with a dedicated TOFU handshake

When the pin does not exist, open a bounded TLS connection to the URL's authority with normal hostname/chain verification disabled for that handshake only. Do not send a client certificate or any HTTP request through this untrusted connection. Capture and parse the presented leaf certificate, close the connection, and persist the leaf in PEM form. The normal Incus client then reconnects with that PEM as its server certificate pin and the configured client identity.

Alternative: let the Incus client connect with verification disabled. Rejected because it would combine first-use acquisition with authenticated API traffic and make it easier for insecure verification to leak into subsequent operations.

Alternative: shell out to `openssl s_client`. Rejected because native Go TLS avoids a runtime dependency, handles IPv6 URL authorities correctly, and is directly testable.

### D3: Persist without clobbering

Write a complete temporary PEM in the destination directory, sync and close it, then publish it with a no-clobber operation. If another process wins the race, discard the candidate and read the already-persisted pin. Existing unreadable or malformed files are errors, never enrollment triggers.

Alternative: ordinary rename. Rejected because rename can replace an existing pin and violates fail-closed concurrency semantics.

### D4: Existing files are authoritative

If the effective pin exists, read and validate it before connecting. Never contact the target to refresh it. A target reinstall or certificate rotation therefore produces a TLS failure until an operator deliberately verifies and removes/replaces the pin.

### D5: Report first-use trust without leaking credentials

Connection metadata records whether TOFU occurred plus the pin path and colon-delimited SHA-256 leaf fingerprint. `doctor` prints this as a first-use trust event; `run` emits a structured warning. No certificate body, private key, or target URL is logged.

## Risks / Trade-offs

- **[Initial connection can be intercepted]** → The feature is explicitly documented and visibly reports the persisted fingerprint for out-of-band checking.
- **[Legitimate certificate rotation causes downtime]** → Fail closed; document deliberate removal/replacement rather than automatic rotation.
- **[A stale default pin blocks a new URL]** → Error identifies the pin path; changing the URL never silently changes trust.
- **[Two processes bootstrap simultaneously]** → No-clobber publication ensures only one candidate becomes authoritative; losers load the winner.
- **[A process crashes while preparing the certificate]** → Temporary files are not authoritative; only a complete persisted pin is used.

## Migration Plan

1. Existing configurations with `target.cert_path` continue using that path unchanged.
2. Existing configurations without it begin TOFU on the next `run` or `doctor` and create `target.crt` beside the config.
3. Rollback is the previous binary; the created PEM remains usable by explicitly setting `target.cert_path` to it.
