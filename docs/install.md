# Install

Build a static binary on any machine with Go:

```
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o scimitar ./cmd/scimitar
```

For another CPU, set `GOARCH`, for example `GOARCH=arm64`.

On the box that will run Scimitar:

```
sudo install -m 755 scimitar /usr/local/bin/scimitar
sudo install -d -m 700 /etc/scimitar
sudo install -m 600 scimitar.json /etc/scimitar/scimitar.json
sudo install -m 644 deploy/scimitar.service /etc/systemd/system/scimitar.service
sudo systemctl daemon-reload
sudo systemctl enable --now scimitar
```

In `scimitar.json`, set `"data_dir": "/var/lib/scimitar"` and set `listen` to the address you want, for example `"0.0.0.0:8080"` for your LAN. See [config.md](config.md).

The service runs as a throwaway system user with no access to your home directories and a read-only system. Only the service can read the config, which systemd hands to it at start.

Get the admin password and the certificate fingerprint from the first start:

```
sudo journalctl -u scimitar | grep -E 'Admin password|cert_sha256'
```

To reset the password:

```
sudo systemctl stop scimitar
sudo scimitar -config /etc/scimitar/scimitar.json reset-password
sudo chown -R --reference=/var/lib/private/scimitar /var/lib/private/scimitar
sudo systemctl start scimitar
```
