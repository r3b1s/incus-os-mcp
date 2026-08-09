## MODIFIED Requirements

### Requirement: Server and cluster state
- The server SHALL provide tools that report server info and cluster membership/state, and that perform supported cluster-member evacuation and restoration actions.
#### Scenario: an-agent-inspects-server/cluster-state
- **WHEN** an agent inspects server or cluster state
- **THEN** it receives the same server, member, and member-state information as the corresponding API endpoints.
- **WHEN** an agent requests evacuation or restoration of a cluster member
- **THEN** the requested action and optional mode are sent to the API and the resulting operation is reported.

### Requirement: IncusOS system management
- The server SHALL manage the IncusOS appliance layer through the configured admin credential: OS update status/configuration, installed applications, and security information (including encryption recovery keys).
#### Scenario: an-agent-queries-OS-update/applications/security-state
- **WHEN** an agent queries OS update, applications, or security state
- **THEN** the request is authenticated with the configured admin credential and receives the current status from the IncusOS system API.
- **WHEN** an agent triggers an OS update or application change
- **THEN** the request is authenticated with the configured admin credential, the change is applied per the IncusOS system API semantics, and its result is reported.
- **NOTE**: The appliance API is proxied through the Incus application at `/os/1.0/...`; debug endpoints (`/1.0/debug/*`) and internal endpoints (`/internal/*`) remain explicitly outside the tool surface.
