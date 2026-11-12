package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/elliotnorth/Scimitar/deploy"
	"github.com/elliotnorth/Scimitar/internal/auth"
	"github.com/elliotnorth/Scimitar/internal/providers/proxmox"
)

const defaultWebPort = 8080

type installer struct {
	prompter
	system
	readSecret func(prompter, string) (string, error)
}

func (i installer) install(ctx context.Context) error {
	i.say("Scimitar %s setup\n", version)
	reinstall := fileExists(i.paths.configFile())
	if reinstall {
		replace, err := i.confirm("Scimitar is already installed. Replace its settings?", false)
		if err != nil || !replace {
			return errAborted
		}
	}
	pve, err := i.askProxmox(ctx)
	if err != nil {
		return err
	}
	listen, err := i.askListen(reinstall)
	if err != nil {
		return err
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
	_, port, _ := net.SplitHostPort(listen)
	if err := i.healthy(ctx, port); err != nil {
		i.say("\nScimitar was installed but is not answering yet: %v", err)
		i.say("See what happened with: journalctl -u scimitar -n 30")
		return errAborted
	}
	i.printDone(listen, password, certFP)
	return nil
}

func (i installer) askProxmox(ctx context.Context) (proxmox.Config, error) {
	haveToken, err := i.confirm("Do you already have a read-only Proxmox API token for Scimitar?", false)
	if err != nil {
		return proxmox.Config{}, err
	}
	if !haveToken {
		i.showTokenSteps()
		if _, err := i.ask("Press Enter when you are ready", ""); err != nil {
			return proxmox.Config{}, err
		}
	}
	pveURL, fingerprint, err := i.askServer(ctx)
	if err != nil {
		return proxmox.Config{}, err
	}
	for {
		cfg, err := i.askToken(pveURL, fingerprint)
		if err != nil {
			return proxmox.Config{}, err
		}
		i.say("Checking the connection...")
		count, err := testProxmox(ctx, cfg)
		if err == nil {
			i.say("Connected. Found %d %s and %d VMs and containers.\n", count.hosts, plural(count.hosts, "host", "hosts"), count.guests)
			return cfg, nil
		}
		i.say("Proxmox did not accept the connection: %v", err)
		if retry, err := i.confirm("Try another token?", true); err != nil || !retry {
			return proxmox.Config{}, errAborted
		}
	}
}

func (i installer) showTokenSteps() {
	i.say("\nRun these three commands in a shell on your Proxmox host, then come back:\n")
	i.say("  pveum user add scimitar@pve")
	i.say("  pveum user token add scimitar@pve scimitar --privsep 1")
	i.say("  pveum acl modify / --tokens 'scimitar@pve!scimitar' --roles PVEAuditor\n")
	i.say("The second command prints a secret (the \"value\" line). Keep it for the next step.\n")
}

func (i installer) askServer(ctx context.Context) (pveURL, fingerprint string, err error) {
	for {
		address, err := i.required("Proxmox address (for example 192.168.1.10)")
		if err != nil {
			return "", "", err
		}
		pveURL, err = proxmoxURL(address)
		if err == nil {
			fingerprint, err = fetchFingerprint(ctx, pveURL)
		}
		if err != nil {
			i.say("%v. Check the address and try again.", err)
			continue
		}
		i.say("\nProxmox certificate fingerprint:\n  %s", fingerprint)
		i.say("You can compare it in the Proxmox web UI under your node, System, Certificates.")
		trusted, err := i.confirm("Trust this certificate?", true)
		if err != nil || !trusted {
			return "", "", errAborted
		}
		return pveURL, fingerprint, nil
	}
}

func (i installer) askToken(pveURL, fingerprint string) (proxmox.Config, error) {
	tokenID, err := i.ask("Token ID", "scimitar@pve!scimitar")
	if err != nil {
		return proxmox.Config{}, err
	}
	secret, err := i.readSecret(i.prompter, "Token secret")
	if err != nil {
		return proxmox.Config{}, err
	}
	return proxmox.Config{URL: pveURL, TokenID: tokenID, TokenSecret: secret, Fingerprint: fingerprint}, nil
}

func (i installer) askListen(reinstall bool) (string, error) {
	openToLAN, err := i.confirm("Allow other devices on your network to open Scimitar?", true)
	if err != nil {
		return "", err
	}
	host := "127.0.0.1"
	if openToLAN {
		host = "0.0.0.0"
	}
	port := defaultWebPort
	for {
		listen := net.JoinHostPort(host, strconv.Itoa(port))
		if reinstall || i.portFree(listen) {
			return listen, nil
		}
		answer, err := i.ask(fmt.Sprintf("Port %d is already in use. Which port should Scimitar use?", port), strconv.Itoa(port+1))
		if err != nil {
			return "", err
		}
		if p, err := strconv.Atoi(answer); err == nil && p > 0 && p < 65536 {
			port = p
		}
	}
}

func (i installer) writeConfig(listen string, pve proxmox.Config) error {
	settings, err := json.Marshal(pve)
	if err != nil {
		return err
	}
	providers := []providerConfig{{Type: "proxmox", Settings: settings}}
	// Reinstalling replaces the Proxmox settings but keeps devices added with add-device.
	if old, err := loadConfig(i.paths.configFile()); err == nil {
		for _, p := range old.Providers {
			if p.Type != "proxmox" {
				providers = append(providers, p)
			}
		}
	}
	cfg := config{
		Listen:          listen,
		DataDir:         i.paths.dataDir,
		DiscoverSeconds: 300,
		StatusSeconds:   30,
		Providers:       providers,
	}
	return saveConfig(i.paths.configFile(), cfg)
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
	if fileExists(i.paths.dataDir.sshKey()) {
		owned = append(owned, i.paths.dataDir.sshKey())
	}
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

func (i installer) printDone(listen, password, certFP string) {
	host, port, _ := net.SplitHostPort(listen)
	if ip := primaryIP(); host == "0.0.0.0" && ip != "" {
		host = ip
	}
	i.say("\nScimitar is running.\n")
	i.say("  Open:      https://%s", net.JoinHostPort(host, port))
	if password != "" {
		i.say("  Password:  %s", password)
		i.say("             Write it down. It is shown only once.")
	} else {
		i.say("  Password:  unchanged from your previous install")
	}
	i.say("\nYour browser will warn about the certificate because Scimitar made its own.")
	i.say("It is safe to continue if the browser shows this fingerprint:\n  %s", certFP)
	i.say("\nUseful commands:")
	i.say("  scimitar status                 see if it is running")
	i.say("  sudo scimitar update            get the newest version")
	i.say("  sudo scimitar reset-password    forgot the password")
	i.say("  sudo scimitar uninstall         remove it")
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
