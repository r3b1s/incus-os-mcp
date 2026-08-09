## MODIFIED Requirements

### Requirement: Network management
- The server SHALL provide project-aware tools to list, create, update, delete, and inspect managed networks (bridge, OVN, physical, macvlan, and other target-supported types).
#### Scenario: an-agent-lists-networks
- **WHEN** an agent lists, creates, updates, or deletes a managed network with a project override
- **THEN** the operation is scoped to that project, and the requested network type/config is returned or applied; target errors are surfaced.

### Requirement: Network ACLs
- The server SHALL provide project-aware tools to list, create, update, and delete ACLs and their rules.
#### Scenario: an-agent-manages-an-ACL
- **WHEN** an agent manages an ACL with a project override
- **THEN** ACL rules are applied or returned in that effective project.

### Requirement: Network zones
- The server SHALL provide project-aware tools to list, create, update, and delete DNS zones; and to list, inspect, create, update, and delete DNS records within those zones.
#### Scenario: an-agent-manages-a-zone
- **WHEN** an agent manages a zone or its records with a project override
- **THEN** zone and record changes are applied in that effective project.
- **WHEN** an agent lists or fetches a DNS record
- **THEN** the tool returns the API record name, description, and entries for the selected zone.
