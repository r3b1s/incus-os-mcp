# Bootstrap: scoped certificate for the MCP server

The MCP server works best under a **restricted (non-admin) certificate**
enrolled in Incus's fine-grained authorization. This page is the operator
runbook for enabling that on the target and enrolling the server certificate.

> Public project: all examples use documentation-only placeholder values.
> Commands marked **(target host)** run on the IncusOS/Incus host itself,
> with an existing admin identity.

## 1. Enable fine-grained authorization

Incus supports two mechanisms (see `incus auth --help` on the target):

- **OpenFGA-based** authorization (the full model), or
- **scriptlet-based** authorization (Incus 7+; simpler rules).

Check what the target supports:

```sh
incus auth show
```

Enable the model you want, then create an **auth group** for the MCP server:

```sh
# OpenFGA example
incus auth group create mcp-server

# Scriptlet example (Incus 7+)
incus auth group create mcp-server --scriptlet '
allow: true
'
```

## 2. Scope the group's permissions

Give the group CRUD on the resource domains the agent needs, and read-only
access to server/cluster state. Certificate mutation and the appliance
(IncusOS system) surface stay admin-only.

```sh
# Examples (adjust to your authorization model)
incus auth group permission add mcp-server instances create,list,update,delete
incus auth group permission add mcp-server images create,list,update,delete
incus auth group permission add mcp-server storage create,list,update,delete
incus auth group permission add mcp-server networks create,list,update,delete
incus auth group permission add mcp-server profiles create,list,update,delete
incus auth group permission add mcp-server server list,get          # read-only
incus auth group permission add mcp-server cluster list,get          # read-only
```

## 3. Mint and enroll the server certificate

On the MCP server host:

```sh
incus-os-mcp cert setup --dir ~/.config/incus-os-mcp
# writes ~/.config/incus-os-mcp/mcp-server.crt + .key (key 0600)
```

On the target host, trust it **scoped to the project and the group**:

```sh
incus config trust add ~/.config/incus-os-mcp/mcp-server.crt \
  --type client \
  --restricted \
  --projects default \
  --group mcp-server
```

The certificate is now revocable independently:

```sh
incus config trust remove <fingerprint>
```

> **Never** reuse a human admin certificate for the MCP server. The server
> inherits exactly the permissions of the certificate it presents; a scoped
> cert limits blast radius, and a dedicated cert can be rotated/revoked
> without touching human access.

## 4. Wire the config

Set `credential.cert_path` / `credential.key_path` to the minted pair. For a
self-signed target, also set `target.cert_path` to the target's TLS
certificate (the server pins it).

## 5. (Optional) full-admin credential

Admin-only surfaces — certificate management and the IncusOS appliance tools
(OS updates, applications, recovery keys) — need a second identity:

```sh
incus-os-mcp cert setup --name mcp-admin --dir ~/.config/incus-os-mcp
incus config trust add ~/.config/incus-os-mcp/mcp-admin.crt --type client
```

Then set `admin_credential.cert_path` / `admin_credential.key_path` in the
config. Without it, those tools report that admin credentials are required
(they degrade cleanly, never crash).

## 6. Verify

```sh
incus-os-mcp doctor
```

All four checks (config, credentials, API reachability, effective
permissions) should pass. With a scoped cert, `doctor`'s admin probe will
report a permission denial if the group lacks certificate read — that is the
expected scoped behavior, not an error.
