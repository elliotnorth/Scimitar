package main

import (
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateCertCreatesOnceAndReuses(t *testing.T) {
	dir := dataDir(t.TempDir())
	first, err := loadOrCreateCert(dir.cert(), dir.key())
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadOrCreateCert(dir.cert(), dir.key())
	if err != nil {
		t.Fatal(err)
	}
	if certFingerprint(first) != certFingerprint(second) {
		t.Fatal("certificate regenerated on second start")
	}
	info, _ := os.Stat(dir.key())
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("key file mode %v", info.Mode().Perm())
	}
	leaf, err := x509.ParseCertificate(first.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := leaf.VerifyHostname("localhost"); err != nil {
		t.Error(err)
	}
	if !containsIP(leaf.IPAddresses, net.IPv4(127, 0, 0, 1)) {
		t.Errorf("127.0.0.1 missing from %v", leaf.IPAddresses)
	}
}

func TestLoadOrCreateCertRejectsBrokenFiles(t *testing.T) {
	dir := t.TempDir()
	cert, key := filepath.Join(dir, "cert.pem"), filepath.Join(dir, "key.pem")
	os.WriteFile(cert, []byte("garbage"), 0o644)
	os.WriteFile(key, []byte("garbage"), 0o600)
	if _, err := loadOrCreateCert(cert, key); err == nil {
		t.Fatal("broken certificate accepted")
	}
}

func containsIP(ips []net.IP, want net.IP) bool {
	for _, ip := range ips {
		if ip.Equal(want) {
			return true
		}
	}
	return false
}
