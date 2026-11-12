package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"time"
)

const (
	serviceUser   = "scimitar"
	healthTimeout = 20 * time.Second
)

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
	portFree   func(listen string) bool
	healthy    func(ctx context.Context, port string) error
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
		portFree:   portFree,
		healthy:    waitHealthy,
	}
}

func portFree(listen string) bool {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return false
	}
	l.Close()
	return true
}

func waitHealthy(ctx context.Context, port string) error {
	client := &http.Client{
		Timeout: 2 * time.Second,
		// Scimitar's own certificate was just created on this machine, so there is nothing to verify it against yet.
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	ctx, cancel := context.WithTimeout(ctx, healthTimeout)
	defer cancel()
	var lastErr error
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://127.0.0.1:"+port+"/", nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return lastErr
		case <-time.After(500 * time.Millisecond):
		}
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
