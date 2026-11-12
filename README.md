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

## Install

```
sudo ./scimitar install
```

See [docs/install.md](docs/install.md). Settings are in [docs/config.md](docs/config.md), Proxmox token details in [docs/proxmox.md](docs/proxmox.md).

## Develop

```
./scripts/setup-hooks.sh
go test ./...
go run ./cmd/scimitar -config scimitar.json
```

## License

Apache-2.0
