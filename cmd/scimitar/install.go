package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/elliotnorth/Scimitar/deploy"
	"github.com/elliotnorth/Scimitar/internal/auth"
	"github.com/elliotnorth/Scimitar/internal/providers/proxmox"
)

const webPort = "8080"

type installer struct {
	prompter
	system
	readSecret func(prompter, string) (string, error)
}

func (i installer) install(ctx context.Context) error {
	i.say("Scimitar setup\n")
	if fileExists(i.paths.configFile()) {
		replace, err := i.confirm("Scimitar is already installed. Replace its settings?", false)
		if err != nil || !replace {
			return errAborted
		}
	}
	pve, err := i.askProxmox(ctx)
	if err != nil {
		return err
	}
	openToLAN, err := i.confirm("Allow other devices on your network to open Scimitar?", true)
	if err != nil {
		return err
	}
	listen := "127.0.0.1:" + webPort
	if openToLAN {
		listen = "0.0.0.0:" + webPort
	}

	i.say("\nInstalling...")
	if err := i.ensureUser(); err != nil {
		return fmt.Errorf("create %s user: %w", serviceUser, err)
	}
	if err := i.installBinary(); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	if err := i.writeConfig(listen, pve); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	password, certFP, err := i.prepareData()
	if err != nil {
		return err
	}
	if err := i.startService(); err != nil {
		return err
	}
	i.printDone(openToLAN, password, certFP)
	return nil
}

func (i installer) askProxmox(ctx context.Context) (proxmox.Config, error) {
	haveToken, err := i.confirm("Do you already have a read-only Proxmox API token for Scimitar?", false)
	if err != nil {
		return proxmox.Config{}, err
	}
	if !haveToken {
		i.say("\nRun these three commands in a shell on your Proxmox host, then come back:\n")
		i.say("  pveum user add scimitar@pve")
		i.say("  pveum user token add scimitar@pve scimitar --privsep 1")
		i.say("  pveum acl modify / --tokens 'scimitar@pve!scimitar' --roles PVEAuditor\n")
		i.say("The second command prints a secret (the \"value\" line). Keep it for the next step.\n")
		if _, err := i.ask("Press Enter when you are ready", ""); err != nil {
			return proxmox.Config{}, err
		}
	}

	address, err := i.required("Proxmox address (for example 192.168.1.10)")
	if err != nil {
		return proxmox.Config{}, err
	}
	pveURL, err := proxmoxURL(address)
	if err != nil {
		return proxmox.Config{}, err
	}
	fingerprint, err := fetchFingerprint(ctx, pveURL)
	if err != nil {
		return proxmox.Config{}, err
	}
	i.say("\nProxmox certificate fingerprint:\n  %s", fingerprint)
	i.say("You can compare it in the Proxmox web UI under your node, System, Certificates.")
	trusted, err := i.confirm("Trust this certificate?", true)
	if err != nil || !trusted {
		return proxmox.Config{}, errAborted
	}

	tokenID, err := i.ask("Token ID", "scimitar@pve!scimitar")
	if err != nil {
		return proxmox.Config{}, err
	}
	secret, err := i.readSecret(i.prompter, "Token secret")
	if err != nil {
		return proxmox.Config{}, err
	}
	cfg := proxmox.Config{URL: pveURL, TokenID: tokenID, TokenSecret: secret, Fingerprint: fingerprint}

	i.say("Checking the connection...")
	count, err := testProxmox(ctx, cfg)
	if err != nil {
		return proxmox.Config{}, fmt.Errorf("Proxmox did not accept the connection: %w", err)
	}
	i.say("Connected. Found %d %s and %d VMs and containers.\n", count.hosts, plural(count.hosts, "host", "hosts"), count.guests)
	return cfg, nil
}

func (i installer) writeConfig(listen string, pve proxmox.Config) error {
	settings, err := json.Marshal(pve)
	if err != nil {
		return err
	}
	cfg := config{
		Listen:          listen,
		DataDir:         i.paths.dataDir,
		DiscoverSeconds: 300,
		StatusSeconds:   30,
		Providers:       []providerConfig{{Type: "proxmox", Settings: settings}},
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(i.paths.configDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(i.paths.configFile(), append(data, '\n'), 0o600)
}

func (i installer) prepareData() (password, certFP string, err error) {
	if err := i.paths.dataDir.ensure(); err != nil {
		return "", "", err
	}
	if !fileExists(i.paths.dataDir.auth()) {
		if password, err = auth.Reset(i.paths.dataDir.auth()); err != nil {
			return "", "", fmt.Errorf("create password: %w", err)
		}
	}
	cert, err := loadOrCreateCert(i.paths.dataDir.cert(), i.paths.dataDir.key())
	if err != nil {
		return "", "", err
	}
	owned := []string{string(i.paths.dataDir), i.paths.dataDir.auth(), i.paths.dataDir.cert(), i.paths.dataDir.key()}
	if err := i.chownToService(owned...); err != nil {
		return "", "", fmt.Errorf("set data ownership: %w", err)
	}
	return password, certFingerprint(cert), nil
}

func (i installer) startService() error {
	if err := os.WriteFile(i.paths.unit, []byte(deploy.SystemdUnit), 0o644); err != nil {
		return fmt.Errorf("write service: %w", err)
	}
	for _, args := range [][]string{{"daemon-reload"}, {"enable", "scimitar"}, {"restart", "scimitar"}} {
		if err := i.run("systemctl", args...); err != nil {
			return err
		}
	}
	return nil
}

func (i installer) printDone(openToLAN bool, password, certFP string) {
	host := "127.0.0.1"
	if ip := primaryIP(); openToLAN && ip != "" {
		host = ip
	}
	i.say("\nScimitar is running.\n")
	i.say("  Open:      https://%s", net.JoinHostPort(host, webPort))
	if password != "" {
		i.say("  Password:  %s", password)
		i.say("             Write it down. It is shown only once.")
	} else {
		i.say("  Password:  unchanged from your previous install")
	}
	i.say("\nYour browser will warn about the certificate because Scimitar made its own.")
	i.say("It is safe to continue if the browser shows this fingerprint:\n  %s", certFP)
	i.say("\nForgot the password? Run: sudo scimitar reset-password")
}

func primaryIP() string {
	for _, ip := range localIPs() {
		if !ip.IsLoopback() && ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
