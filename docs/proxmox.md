# Proxmox setup

Scimitar reads Proxmox through an API token with the read-only `PVEAuditor` role.

On a Proxmox node:

```
pveum user add scimitar@pve
pveum user token add scimitar@pve scimitar --privsep 1
pveum acl modify / --tokens 'scimitar@pve!scimitar' --roles PVEAuditor
```

Keep the secret it prints. It is shown once.

## Certificate

Proxmox uses a self-signed certificate by default. Pin its SHA-256 fingerprint:

```
pvenode cert info --output-format json | grep fingerprint
```

Use the fingerprint of `pveproxy-ssl.pem` if present, otherwise `pve-ssl.pem`. If your certificate is signed by your own CA, give Scimitar the CA file instead.

## What Scimitar reads

| Endpoint | Used for |
|---|---|
| `/nodes` | hosts and their status |
| `/cluster/resources?type=vm` | VMs, containers, status, tags |
| `/nodes/{node}/{qemu,lxc}/{vmid}/config` | notes, used as service hints |

Guest names, tags and notes become service hints. Templates are skipped.
