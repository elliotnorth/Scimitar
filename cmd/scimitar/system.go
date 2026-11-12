package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
)

const serviceUser = "scimitar"

type installPaths struct {
	binary    string
	configDir string
	unit      string
	dataDir   dataDir
}

func (p installPaths) configFile() string {
	return filepath.Join(p.configDir, "scimitar.json")
}

var systemPaths = installPaths{
	binary:    "/usr/local/bin/scimitar",
	configDir: "/etc/scimitar",
	unit:      "/etc/systemd/system/scimitar.service",
	dataDir:   "/var/lib/scimitar",
}

type system struct {
	paths      installPaths
	run        func(name string, args ...string) error
	owner      func() (uid, gid int, err error)
	executable func() (string, error)
}

func realSystem() system {
	return system{
		paths: systemPaths,
		run: func(name string, args ...string) error {
			out, err := exec.Command(name, args...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s: %w: %s", name, err, out)
			}
			return nil
		},
		owner:      lookupServiceUser,
		executable: os.Executable,
	}
}

func lookupServiceUser() (int, int, error) {
	u, err := user.Lookup(serviceUser)
	if err != nil {
		return 0, 0, err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return 0, 0, err
	}
	gid, err := strconv.Atoi(u.Gid)
	return uid, gid, err
}

func (s system) ensureUser() error {
	if _, _, err := s.owner(); err == nil {
		return nil
	}
	return s.run("useradd", "--system", "--no-create-home", "--home-dir", string(s.paths.dataDir), "--shell", "/usr/sbin/nologin", serviceUser)
}

func (s system) installBinary() error {
	self, err := s.executable()
	if err != nil {
		return err
	}
	if same, _ := sameFile(self, s.paths.binary); same {
		return nil
	}
	src, err := os.Open(self)
	if err != nil {
		return err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(filepath.Dir(s.paths.binary), ".scimitar-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.paths.binary)
}

func sameFile(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(infoA, infoB), nil
}

func (s system) chownToService(paths ...string) error {
	uid, gid, err := s.owner()
	if err != nil {
		return err
	}
	for _, p := range paths {
		if err := os.Chown(p, uid, gid); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
