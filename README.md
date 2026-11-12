# Scimitar

A self-hosted command center for your homelab. One page that shows your hosts, VMs, devices and services, and whether each one is up.

Version 1 connects to Proxmox VE read-only through a limited API token.

## Layout

```
cmd/scimitar/         entry point
internal/domain/      Device, Service, Link, Status, Provider interface
internal/core/        inventory, service matching, review queue
internal/providers/   one package per data source (proxmox first)
internal/api/         HTTP handlers
internal/auth/        admin password and sessions
web/                  UI, embedded into the binary
deploy/               systemd service
docs/                 short guides
hooks/                git hooks
scripts/              setup scripts
```

## Run

```
go build -o scimitar ./cmd/scimitar
./scimitar -config scimitar.json
```

Then open https://127.0.0.1:8080 and log in with the admin password printed on first start.

See [docs/config.md](docs/config.md), [docs/proxmox.md](docs/proxmox.md) and [docs/install.md](docs/install.md).

## Develop

```
./scripts/setup-hooks.sh
go test ./...
```

## License

Apache-2.0
