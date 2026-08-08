## Purpose

Expose Incus network management as MCP tools: managed networks, network ACLs, and network zones.

## ADDED Requirements

### Requirement: Network management
- The server SHALL tools list, create, update, delete, and inspect managed networks (bridge, OVN, physical, macvlan, etc.).
#### Scenario: an-agent-lists-networks
- **WHEN** an agent lists networks
- **THEN** it receives network names, types, and config.
- **WHEN** an agent creates/updates/deletes a network
- **THEN** the network is created with the requested type/config or the error is surfaced; deletion fails when networks are in use.

### Requirement: Network ACLs
- The server SHALL tools list, create, update, delete ACLs and their rules.
#### Scenario: an-agent-manages-an-ACL
- **WHEN** an agent manages an ACL
- **THEN** ACL rules are applied per the API.

### Requirement: Network zones
- The server SHALL tools list, create, update, delete DNS zones and their records.
#### Scenario: an-agent-manages-a-zone
- **WHEN** an agent manages a zone
- **THEN** zone and record changes are applied per the API.
