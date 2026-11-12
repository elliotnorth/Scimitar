package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeReleases(t *testing.T, tag string, binary []byte, checksum string) releases {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+releaseRepo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	})
	mux.HandleFunc("/"+releaseRepo+"/releases/download/"+tag+"/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(checksum + "  " + assetName() + "\nffff  scimitar-linux-other\n"))
	})
	mux.HandleFunc("/"+releaseRepo+"/releases/download/"+tag+"/"+assetName(), func(w http.ResponseWriter, r *http.Request) {
		w.Write(binary)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return releases{apiBase: srv.URL, downloadBase: srv.URL, client: srv.Client()}
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func updateTarget(t *testing.T, withUnit bool) (system, *[]string) {
	t.Helper()
	dir := t.TempDir()
	var commands []string
	s := system{
		paths: installPaths{binary: filepath.Join(dir, "scimitar"), unit: filepath.Join(dir, "scimitar.service")},
		run: func(name string, args ...string) error {
			commands = append(commands, name+" "+strings.Join(args, " "))
			return nil
		},
	}
	os.WriteFile(s.paths.binary, []byte("old"), 0o755)
	if withUnit {
		os.WriteFile(s.paths.unit, []byte("unit"), 0o644)
	}
	return s, &commands
}

func TestUpdateReplacesBinaryAndRestarts(t *testing.T) {
	newBinary := []byte("new version")
	s, commands := updateTarget(t, true)
	out := &bytes.Buffer{}
	u := updater{releases: fakeReleases(t, "v9.9.9", newBinary, sha(newBinary)), system: s, out: out}
	if err := u.update(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(s.paths.binary)
	info, _ := os.Stat(s.paths.binary)
	if string(got) != "new version" || info.Mode().Perm() != 0o755 {
		t.Fatalf("binary %q mode %v", got, info.Mode().Perm())
	}
	if strings.Join(*commands, ";") != "systemctl restart scimitar" {
		t.Fatalf("commands: %v", *commands)
	}
	if !strings.Contains(out.String(), "to v9.9.9") {
		t.Fatalf("output: %s", out)
	}
}

func TestUpdateRejectsBadChecksum(t *testing.T) {
	s, commands := updateTarget(t, true)
	u := updater{releases: fakeReleases(t, "v9.9.9", []byte("tampered"), sha([]byte("original"))), system: s, out: &bytes.Buffer{}}
	if err := u.update(context.Background()); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("got %v", err)
	}
	if got, _ := os.ReadFile(s.paths.binary); string(got) != "old" || len(*commands) != 0 {
		t.Fatal("binary changed despite bad checksum")
	}
}

func TestUpdateWhenAlreadyCurrent(t *testing.T) {
	s, commands := updateTarget(t, true)
	out := &bytes.Buffer{}
	u := updater{releases: fakeReleases(t, version, nil, ""), system: s, out: out}
	if err := u.update(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "already the newest") || len(*commands) != 0 {
		t.Fatalf("output %s commands %v", out, *commands)
	}
}

func TestUpdateWithoutServiceSkipsRestart(t *testing.T) {
	b := []byte("new")
	s, commands := updateTarget(t, false)
	u := updater{releases: fakeReleases(t, "v9.9.9", b, sha(b)), system: s, out: &bytes.Buffer{}}
	if err := u.update(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(*commands) != 0 {
		t.Fatalf("commands: %v", *commands)
	}
}

func TestChecksumFor(t *testing.T) {
	sums := []byte("aaaa  scimitar-linux-amd64\nbbbb *scimitar-linux-arm64\n")
	if got, _ := checksumFor(sums, "scimitar-linux-arm64"); got != "bbbb" {
		t.Errorf("got %q", got)
	}
	if _, err := checksumFor(sums, "scimitar-linux-riscv64"); err == nil {
		t.Error("missing asset accepted")
	}
}
