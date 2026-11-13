package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/elliotnorth/Scimitar/internal/providers/junos"
	"github.com/elliotnorth/Scimitar/internal/sshkey"
	"golang.org/x/crypto/ssh"
)

// deviceAdder walks the user through adding a Juniper device to an existing install.
type deviceAdder struct {
	prompter
	system
	configPath   string
	fetchHostKey func(ctx context.Context, address string) (fingerprint, keyType string, err error)
	connect      func(ctx context.Context, cfg junos.Config) (junos.SystemInfo, error)
}

func newDeviceAdder(p prompter, s system, configPath string) deviceAdder {
	return deviceAdder{prompter: p, system: s, configPath: configPath, fetchHostKey: junos.FetchHostKey, connect: connectJunos}
}

func connectJunos(ctx context.Context, cfg junos.Config) (junos.SystemInfo, error) {
	p, err := junos.New(cfg)
	if err != nil {
		return junos.SystemInfo{}, err
	}
	defer p.Close()
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	return p.Info(ctx)
}

func (a deviceAdder) add(ctx context.Context) error {
	cfg, err := loadConfig(a.configPath)
	if err != nil {
		return fmt.Errorf("could not read the Scimitar config (%w). Is Scimitar installed? Run: sudo scimitar install", err)
	}
	if err := cfg.DataDir.ensure(); err != nil {
		return err
	}
	keyPath := cfg.DataDir.sshKey()
	signer, err := sshkey.LoadOrCreate(keyPath)
	if err != nil {
		return err
	}
	if signer.PublicKey().Type() != ssh.KeyAlgoRSA && !hasJunos(cfg) {
		// Keys made before Scimitar switched to RSA are not accepted by older Junos.
		// Nothing uses the key yet, so it is safe to swap.
		if signer, err = sshkey.Replace(keyPath); err != nil {
			return err
		}
	}
	if err := matchOwner(keyPath, string(cfg.DataDir)); err != nil {
		return fmt.Errorf("set ssh key owner: %w", err)
	}

	a.say("Add a Juniper device (EX, SRX, QFX or MX)\n")
	a.say("1. On the device, give Scimitar a read-only login. Paste these lines into its CLI:\n")
	a.say("  configure")
	a.say("  set system login user scimitar class read-only")
	a.say("  set system login user scimitar authentication ssh-rsa \"%s\"", sshkey.AuthorizedKey(signer))
	a.say("  set protocols lldp interface all")
	a.say("  commit and-quit\n")
	a.say("If a pasted line breaks, type \"load set terminal\" after configure, paste the set lines,")
	a.say("press Ctrl+D, then commit and-quit.\n")
	a.say("The read-only class cannot change anything. The lldp line lets Scimitar see what is")
	a.say("plugged into each port; skip it if LLDP is already on.\n")
	if _, err := a.ask("Press Enter when you have committed it", ""); err != nil {
		return err
	}

	address, fingerprint, err := a.askAddress(ctx)
	if err != nil {
		return err
	}
	username, err := a.ask("Username on the device", "scimitar")
	if err != nil {
		return err
	}
	device := junos.Config{Name: "new", Address: address, Username: username, HostKey: fingerprint, KeyFile: keyPath}
	info, err := a.tryLogin(ctx, device)
	if err != nil {
		return err
	}
	a.say("Connected to %s: %s running Junos %s.\n", info.Hostname, strings.ToUpper(info.Model), info.Version)

	device.Name, err = a.askName(info.Hostname, cfg)
	if err != nil {
		return err
	}
	device.KeyFile = "" // the default key path in the data dir is used at startup
	settings, err := json.Marshal(device)
	if err != nil {
		return err
	}
	cfg.Providers = append(cfg.Providers, providerConfig{Type: "junos", Settings: settings})
	if err := saveConfig(a.configPath, cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	if fileExists(a.paths.unit) && a.configPath == a.paths.configFile() {
		if err := a.run("systemctl", "restart", "scimitar"); err != nil {
			return err
		}
		a.say("Added %s. Scimitar restarted and will show it under devices within a minute.", device.Name)
		return nil
	}
	a.say("Added %s. Restart Scimitar to start reading it.", device.Name)
	return nil
}

func hasJunos(cfg config) bool {
	for _, p := range cfg.Providers {
		if p.Type == "junos" {
			return true
		}
	}
	return false
}

func (a deviceAdder) askAddress(ctx context.Context) (address, fingerprint string, err error) {
	for {
		raw, err := a.required("2. Device address (for example 192.168.1.2)")
		if err != nil {
			return "", "", err
		}
		address, err = junos.NormalizeAddress(raw)
		if err == nil {
			var keyType string
			fingerprint, keyType, err = a.fetchHostKey(ctx, address)
			if err == nil {
				a.say("\nThe device's SSH key (%s):\n  %s", keyType, fingerprint)
				a.say("If you have logged in to it before, compare with: ssh-keygen -l -F %s", hostOnly(address))
				trusted, err := a.confirm("Trust this key?", true)
				if err != nil || !trusted {
					return "", "", errAborted
				}
				return address, fingerprint, nil
			}
		}
		a.say("%v. Check the address and try again.", err)
	}
}

func (a deviceAdder) tryLogin(ctx context.Context, device junos.Config) (junos.SystemInfo, error) {
	for {
		a.say("Logging in...")
		info, err := a.connect(ctx, device)
		if err == nil {
			return info, nil
		}
		a.say("Could not log in: %v", err)
		a.say("Check that the lines from step 1 were committed on the device.")
		if retry, err := a.confirm("Try again?", true); err != nil || !retry {
			return junos.SystemInfo{}, errAborted
		}
	}
}

func (a deviceAdder) askName(hostname string, cfg config) (string, error) {
	taken := map[string]bool{}
	for _, p := range cfg.Providers {
		taken[providerName(p)] = true
	}
	suggestion := deviceNameFrom(hostname)
	for {
		name, err := a.ask("Name in Scimitar", suggestion)
		if err != nil {
			return "", err
		}
		switch {
		case !junos.ValidName(name):
			a.say("Use letters, numbers, dots, dashes or underscores.")
		case taken[name]:
			a.say("%s is already used. Pick another name.", name)
		default:
			return name, nil
		}
	}
}

var invalidNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func deviceNameFrom(hostname string) string {
	name := strings.Trim(invalidNameChars.ReplaceAllString(hostname, "-"), "-._")
	if len(name) > 63 {
		name = name[:63]
	}
	if !junos.ValidName(name) {
		return "juniper"
	}
	return name
}

func hostOnly(address string) string {
	if i := strings.LastIndex(address, ":22"); i > 0 && i == len(address)-3 {
		return strings.TrimSuffix(strings.TrimPrefix(address[:i], "["), "]")
	}
	return address
}

func printSSHKey(configPath string) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("%w (try: sudo scimitar ssh-key)", err)
	}
	if err := cfg.DataDir.ensure(); err != nil {
		return err
	}
	signer, err := sshkey.LoadOrCreate(cfg.DataDir.sshKey())
	if err != nil {
		return err
	}
	if err := matchOwner(cfg.DataDir.sshKey(), string(cfg.DataDir)); err != nil {
		return err
	}
	fmt.Println(sshkey.AuthorizedKey(signer))
	return nil
}
