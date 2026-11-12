package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/providers/proxmox"
)

const (
	proxmoxDefaultPort = "8006"
	probeTimeout       = 15 * time.Second
)

func proxmoxURL(address string) (string, error) {
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}
	u, err := url.Parse(address)
	if err != nil || u.Hostname() == "" {
		return "", fmt.Errorf("%q is not a valid address", address)
	}
	if u.Scheme != "https" {
		return "", errors.New("Proxmox must be reached over https")
	}
	port := u.Port()
	if port == "" {
		port = proxmoxDefaultPort
	}
	return "https://" + net.JoinHostPort(u.Hostname(), port), nil
}

func fetchFingerprint(ctx context.Context, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: probeTimeout},
		// The certificate is not trusted yet; its fingerprint is shown to the user to confirm.
		Config: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12},
	}
	conn, err := dialer.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		return "", fmt.Errorf("could not reach %s: %w", u.Host, err)
	}
	defer conn.Close()
	certs := conn.(*tls.Conn).ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return "", errors.New("Proxmox sent no certificate")
	}
	return colonHex(sha256.Sum256(certs[0].Raw)), nil
}

func colonHex(sum [sha256.Size]byte) string {
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":")
}

type discoveryCount struct {
	hosts, guests int
}

func testProxmox(ctx context.Context, cfg proxmox.Config) (discoveryCount, error) {
	p, err := proxmox.New(cfg)
	if err != nil {
		return discoveryCount{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	d, err := p.Discover(ctx)
	if err != nil {
		return discoveryCount{}, err
	}
	var count discoveryCount
	for _, device := range d.Devices {
		if device.Kind == domain.DeviceHost {
			count.hosts++
		} else {
			count.guests++
		}
	}
	return count, nil
}
