## Purpose

Allow a locally built, per-run ISO to enter Incus as a custom ISO volume so disposable virtual machines can consume first-boot configuration without appliance filesystem access.

## ADDED Requirements

### Requirement: Custom ISO volume import
The server SHALL provide a tool that imports a bounded standard-base64 ISO artifact into a named custom ISO storage volume in the requested effective project and storage pool. The tool SHALL reject malformed base64, decoded artifacts larger than 64 MiB, and data lacking an ISO9660 primary-volume descriptor before contacting Incus. It SHALL use only an ephemeral temporary file for the streaming client API and remove it after the request. The tool SHALL report extension, validation, upload, authorization, and asynchronous-operation errors as tool errors, and SHALL wait for the import operation by default.

#### Scenario: an-agent-imports-a-combustion-iso
- **WHEN** an agent supplies a bounded base64 Combustion ISO artifact, a pool, a run-unique volume name, and an effective project
- **THEN** Incus receives a custom ISO volume with that name in the requested project and the tool returns the completed operation result

#### Scenario: the-target-cannot-import-custom-iso-volumes
- **WHEN** the target does not support custom ISO volumes or rejects the supplied file
- **THEN** the tool returns a clear error and does not report a successful import

### Requirement: Imported ISO volumes can be used at VM creation
The imported volume SHALL be usable as the source of a read-only disk device supplied to existing VM creation, without requiring appliance-shell access, a host filesystem path on IncusOS, or an Incus CLI.

#### Scenario: an-agent-creates-a-vm-with-an-imported-iso
- **WHEN** an agent creates a virtual machine with a disk device that references the imported ISO volume and its pool
- **THEN** the VM creation request preserves that device for the VM's first boot
