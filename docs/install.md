# Install

You need a Linux machine with systemd. It can be a small VM or container on Proxmox itself.

## 1. Get the binary

Build it with `./scripts/build.sh` on any machine with Go. You get `dist/scimitar-linux-amd64`, and `dist/scimitar-linux-arm64` for a Raspberry Pi.

Copy it to the machine:

```
scp dist/scimitar-linux-amd64 you@your-box:scimitar
```

## 2. Run the installer

```
sudo ./scimitar install
```

It asks for your Proxmox address and API token, checks the connection, and starts Scimitar. If you have no token yet, it shows the three commands to create a read-only one. At the end it prints the address to open and your admin password.

## Later

| Task | Command |
|---|---|
| New admin password | `sudo scimitar reset-password` |
| Logs | `journalctl -u scimitar` |
| Change settings | run `sudo scimitar install` again |
| Remove | `sudo scimitar uninstall` |

## What the installer sets up

- `/usr/local/bin/scimitar`: the program
- `/etc/scimitar/scimitar.json`: settings, readable by root only. See [config.md](config.md)
- `/var/lib/scimitar`: review decisions, password hash, certificate
- `scimitar.service`: runs as the `scimitar` system user, with a read-only system and no access to home folders
