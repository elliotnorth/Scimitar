package main

import (
	"os"
	"path/filepath"
)

type dataDir string

func (d dataDir) ensure() error {
	return os.MkdirAll(string(d), 0o700)
}

func (d dataDir) decisions() string { return filepath.Join(string(d), "decisions.json") }
func (d dataDir) auth() string      { return filepath.Join(string(d), "auth.json") }
func (d dataDir) cert() string      { return filepath.Join(string(d), "cert.pem") }
func (d dataDir) key() string       { return filepath.Join(string(d), "key.pem") }
func (d dataDir) sshKey() string    { return filepath.Join(string(d), "ssh_key") }
func (d dataDir) devices() string   { return filepath.Join(string(d), "devices.json") }
func (d dataDir) appURLs() string   { return filepath.Join(string(d), "apps.json") }
func (d dataDir) alerts() string    { return filepath.Join(string(d), "alerts.json") }
func (d dataDir) events() string    { return filepath.Join(string(d), "events.json") }
