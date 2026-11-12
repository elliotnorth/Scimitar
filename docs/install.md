# Install

On any Linux machine with systemd (a small VM or container on Proxmox works well):

```
curl -fsSL https://github.com/elliotnorth/Scimitar/releases/latest/download/install.sh | sh
```

The script downloads the right binary for your CPU, checks its checksum and starts the setup. Setup asks for your Proxmox address and API token, checks the connection, and starts Scimitar. If you have no token yet, it shows the three commands to create a read-only one. At the end it prints the address to open and your admin password.

To install a specific version, set `SCIMITAR_VERSION=v0.1.0` before `sh`.

## Commands

| Task | Command |
|---|---|
| Is it running, where do I open it | `scimitar status` |
| Get the newest version | `sudo scimitar update` |
| New admin password | `sudo scimitar reset-password` |
| Change settings | `sudo scimitar install` |
| Logs | `journalctl -u scimitar -f` |
| Remove | `sudo scimitar uninstall` |

## What setup creates

- `/usr/local/bin/scimitar`: the program
- `/etc/scimitar/scimitar.json`: settings, readable by root only. See [config.md](config.md)
- `/var/lib/scimitar`: review decisions, password hash, certificate
- `scimitar.service`: runs as the `scimitar` system user, with a read-only system and no access to home folders

## Without the script

Build with `./scripts/build.sh`, copy `dist/scimitar-linux-amd64` to the machine and run `sudo ./scimitar-linux-amd64 install`.

## Publishing a release

`./scripts/release.sh v0.1.0` runs the tests, builds both binaries with checksums, tags the commit and creates the GitHub release. It needs `gh auth login` once.
