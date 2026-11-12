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
web/                  UI, embedded into the binary
docs/                 short guides
hooks/                git hooks
scripts/              setup scripts
```

## Develop

```
./scripts/setup-hooks.sh
go test ./...
```

## License

Apache-2.0
