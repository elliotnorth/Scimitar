# Configuration

Scimitar reads one JSON file, `/etc/scimitar/scimitar.json` by default. `scimitar install` writes it for you. Pass another path with `-config`.

```json
{
  "listen": "127.0.0.1:8080",
  "data_dir": "/var/lib/scimitar",
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
| `listen` | `127.0.0.1:8080` | address for the web page. Only this machine can reach it by default. Use `0.0.0.0:8080` for your LAN |
| `data_dir` | `scimitar-data` | review decisions, password hash and TLS certificate |
| `discover_seconds` | `300` | how often to rediscover devices and services |
| `status_seconds` | `30` | how often to refresh up/down status |
| `providers` | required | data sources, see below |

The file holds an API token secret. Make it readable only by the user running Scimitar: `chmod 600 scimitar.json`.

## Login and HTTPS

On first start Scimitar prints an admin password once and keeps only its hash in `data_dir/auth.json`. To get a new password, run `sudo scimitar reset-password`.

The page is served over HTTPS only. On first start Scimitar creates a self-signed certificate in `data_dir` and logs its SHA-256 fingerprint. Compare it with the one your browser shows before you accept the certificate. To use your own certificate, replace `cert.pem` and `key.pem` in `data_dir`.

## Providers

`junos`: Juniper EX, SRX, QFX and MX over SSH. Add them with `sudo scimitar add-device`; see [junos.md](junos.md).

`proxmox`: see [proxmox.md](proxmox.md). Use `ca_file` instead of `fingerprint` if your certificate is signed by your own CA. Give each provider a different `name` when you add more than one cluster.

## Service status

A service is `down` when its device is down. Otherwise it is `unknown` until service health checks exist, so a running VM never makes a service look `up` by guesswork.
