package main

import (
	"errors"
	"io/fs"
	"os"
)

func (i installer) uninstall() error {
	remove, err := i.confirm("Remove Scimitar from this machine? Your data folder is kept.", false)
	if err != nil || !remove {
		return errAborted
	}
	if fileExists(i.paths.unit) {
		if err := i.run("systemctl", "disable", "--now", "scimitar"); err != nil {
			return err
		}
	}
	for _, path := range []string{i.paths.unit, i.paths.binary} {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	if err := os.RemoveAll(i.paths.configDir); err != nil {
		return err
	}
	if err := i.run("systemctl", "daemon-reload"); err != nil {
		return err
	}
	i.say("Scimitar is removed.")
	i.say("To also delete its data and user: sudo rm -rf %s && sudo userdel %s", i.paths.dataDir, serviceUser)
	return nil
}
