# Juniper devices

Scimitar reads Juniper EX, SRX, QFX and MX devices over SSH with a read-only login. Each device is added on its own.

## Add a device

On the Scimitar machine:

```
sudo scimitar add-device
```

It prints a few lines to paste into the device's CLI:

```
configure
set system login user scimitar class read-only
set system login user scimitar authentication ssh-ed25519 "ssh-ed25519 AAAA... scimitar"
set protocols lldp interface all
commit and-quit
```

If a long pasted line breaks, type `load set terminal` after `configure`, paste the `set` lines, press Ctrl+D, then `commit and-quit`.

The `read-only` login class cannot change the configuration. The LLDP line lets Scimitar see what is plugged into each port; leave it out if LLDP is already on.

Then it asks for the device's address, shows its SSH host key fingerprint for you to confirm, logs in to check, and asks what to call the device. Scimitar restarts and shows the device within a minute.

To see the public key again: `sudo scimitar ssh-key`.

## Host key

Scimitar remembers the device's host key fingerprint and refuses to connect if it changes. If you replace the device or regenerate its keys, remove its entry from `/etc/scimitar/scimitar.json` and add it again.

## What Scimitar runs

Only these commands, never anything else:

| Command | Used for |
|---|---|
| `show version \| display json` | model, Junos version, up/down status |
| `show interfaces terse \| display json` | ports and their state |
| `show interfaces descriptions \| display json` | port labels |
| `show lldp neighbors \| display json` | what each port is connected to |

## Settings

`add-device` writes one entry per device to the config:

```json
{
  "type": "junos",
  "settings": {
    "name": "ex3400",
    "address": "192.168.1.2:22",
    "username": "scimitar",
    "host_key": "SHA256:..."
  }
}
```

The private key lives in the data directory as `ssh_key`, readable only by the `scimitar` user. Set `key_file` to use a different one.
