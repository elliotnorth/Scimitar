# Configuration

Scimitar reads one JSON file, `scimitar.json` by default. Pass another path with `-config`.

```json
{
  "state_file": "/var/lib/scimitar/decisions.json",
  "discover_seconds": 300,
  "status_seconds": 30,
  "providers": [
    {
      "type": "proxmox",
      "settings": {
        "name": "proxmox",
        "url": "https://pve.lan:8006",
        "token_id": "scimitar@pve!scimitar",
        "token_secret": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "fingerprint": "AB:CD:..."
      }
    }
  ]
}
```

| Field | Default | Meaning |
|---|---|---|
| `state_file` | `scimitar-decisions.json` | where your review decisions are kept |
| `discover_seconds` | `300` | how often to rediscover devices and services |
| `status_seconds` | `30` | how often to refresh up/down status |
| `providers` | required | data sources, see below |

The file holds an API token secret. Make it readable only by the user running Scimitar: `chmod 600 scimitar.json`.

## Providers

`proxmox`: see [proxmox.md](proxmox.md). Use `ca_file` instead of `fingerprint` if your certificate is signed by your own CA. Give each provider a different `name` when you add more than one cluster.

## Service status

A service is `down` when its device is down. Otherwise it is `unknown` until service health checks exist, so a running VM never makes a service look `up` by guesswork.
