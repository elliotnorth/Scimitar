// Package probe checks whether an address answers, without logging in to anything.
package probe

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	checkTimeout = 5 * time.Second
	dialTimeout  = 3 * time.Second
)

// CommonPorts are tried when a device has no port of its own to check.
var CommonPorts = []int{80, 443, 22, 445, 53, 8080}

type Prober struct {
	client *http.Client
}

func New() *Prober {
	return &Prober{client: &http.Client{
		Timeout: checkTimeout,
		Transport: &http.Transport{
			// Homelab apps mostly use self-signed certificates. The check only asks
			// "does it answer" and sends no credentials, so the certificate is not verified.
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
		},
		// A redirect to a login page already proves the app is answering.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// Check reports whether target answers. An http or https target is up when it
// replies with anything but a server error; a tcp target is up when the port opens.
func (p *Prober) Check(ctx context.Context, target string) bool {
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	if u.Scheme == "tcp" {
		return open(ctx, u.Host)
	}
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "scimitar-health-check")
	resp, err := p.client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < http.StatusInternalServerError
}

func open(ctx context.Context, hostPort string) bool {
	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// PortOpen reports whether a TCP port on host accepts connections.
func PortOpen(ctx context.Context, host string, port int) bool {
	return open(ctx, net.JoinHostPort(host, strconv.Itoa(port)))
}

// Reachable reports whether host is on the network, the way a ping would,
// without needing the privileges ping does. A port that opens proves the host
// is up, and so does one that actively refuses: only a live host sends a refusal.
func Reachable(ctx context.Context, host string) bool {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make(chan bool, len(CommonPorts))
	for _, port := range CommonPorts {
		go func() {
			d := net.Dialer{Timeout: dialTimeout}
			conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
			if err == nil {
				conn.Close()
			}
			results <- err == nil || errors.Is(err, syscall.ECONNREFUSED)
		}()
	}
	for range CommonPorts {
		if <-results {
			return true
		}
	}
	return false
}

// NormalizeTarget checks an address typed by the user and returns it in the
// form Check expects. A bare "host:port" becomes an http address.
func NormalizeTarget(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("address is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil {
		return "", fmt.Errorf("%q is not a valid address", raw)
	}
	switch u.Scheme {
	case "http", "https":
	case "tcp":
		if u.Port() == "" {
			return "", errors.New("a tcp address needs a port, like tcp://192.168.1.5:1883")
		}
	default:
		return "", errors.New("address must start with http://, https:// or tcp://")
	}
	if len(raw) > 512 {
		return "", errors.New("address is too long")
	}
	return u.String(), nil
}
