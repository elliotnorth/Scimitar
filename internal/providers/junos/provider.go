// Package junos reads Juniper devices (EX, SRX, QFX, MX) over SSH with a fixed set of read-only commands.
package junos

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"

	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/sshkey"
)

// The only commands this provider runs. Nothing here takes user input.
// XML output works on every Junos release; JSON needs 14.2 or newer.
const (
	cmdVersion      = "show version | display xml"
	cmdPorts        = "show interfaces terse | display xml"
	cmdDescriptions = "show interfaces descriptions | display xml"
	cmdNeighbors    = "show lldp neighbors | display xml"
)

var _ domain.Provider = (*Provider)(nil)

type Provider struct {
	name string
	host string
	conn *conn

	mu   sync.Mutex
	info SystemInfo
}

func New(cfg Config) (*Provider, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	addr, err := NormalizeAddress(cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("junos: %w", err)
	}
	signer, err := sshkey.Load(cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("junos: %w", err)
	}
	host, _, _ := net.SplitHostPort(addr)
	return &Provider{
		name: cfg.Name,
		host: host,
		conn: &conn{
			addr: addr,
			config: withOlderJunos(&ssh.ClientConfig{
				User:            cfg.Username,
				Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
				HostKeyCallback: pinnedHostKey(cfg.HostKey),
				Timeout:         dialTimeout,
			}),
		},
	}, nil
}

func (p *Provider) Name() string { return p.name }

func (p *Provider) deviceID() string { return p.name + "/device" }

// Close ends the SSH connection, if one is open.
func (p *Provider) Close() { p.conn.close() }

// Info runs "show version" and returns the hostname, model and Junos version.
func (p *Provider) Info(ctx context.Context) (SystemInfo, error) {
	out, err := p.conn.run(ctx, cmdVersion)
	if err != nil {
		return SystemInfo{}, err
	}
	info, err := parseVersion(out)
	if err != nil {
		return SystemInfo{}, err
	}
	p.mu.Lock()
	p.info = info
	p.mu.Unlock()
	return info, nil
}

func (p *Provider) Discover(ctx context.Context) (domain.Discovery, error) {
	info, err := p.Info(ctx)
	if err != nil {
		return domain.Discovery{}, err
	}
	neighbors, err := p.neighbors(ctx)
	if err != nil {
		return domain.Discovery{}, err
	}
	id := p.deviceID()
	d := domain.Discovery{Devices: []domain.Device{{
		ID:        id,
		Name:      p.name,
		Kind:      kindFor(info.Model),
		Provider:  p.name,
		Addresses: []string{p.host},
		Status:    domain.StatusUp,
	}}}
	seen := map[string]bool{}
	for _, n := range neighbors {
		if n.System == "" || seen[n.System] {
			continue
		}
		seen[n.System] = true
		d.Links = append(d.Links, domain.Link{From: id, To: domain.NameRef(n.System), Kind: domain.LinkConnected})
	}
	return d, nil
}

// neighbors returns LLDP neighbors. A device without LLDP running answers with
// an error instead of neighbor data; that means no neighbors, not a failure.
func (p *Provider) neighbors(ctx context.Context) ([]Neighbor, error) {
	out, err := p.conn.run(ctx, cmdNeighbors)
	if err != nil {
		return nil, err
	}
	neighbors, err := parseNeighbors(out)
	if err != nil {
		return nil, nil
	}
	return neighbors, nil
}

func (p *Provider) Status(ctx context.Context) (map[string]domain.Status, error) {
	if _, err := p.Info(ctx); err != nil {
		return nil, err
	}
	return map[string]domain.Status{p.deviceID(): domain.StatusUp}, nil
}

func (p *Provider) Details(ctx context.Context, deviceID string) ([]domain.Detail, error) {
	if deviceID != p.deviceID() {
		return nil, fmt.Errorf("junos: device %q does not belong to %s", deviceID, p.name)
	}
	info, err := p.Info(ctx)
	if err != nil {
		return nil, err
	}
	out, err := p.conn.run(ctx, cmdPorts)
	if err != nil {
		return nil, err
	}
	ports, err := parsePorts(out)
	if err != nil {
		return nil, err
	}
	descriptions := map[string]string{}
	if out, err := p.conn.run(ctx, cmdDescriptions); err == nil {
		if d, err := parseDescriptions(out); err == nil {
			descriptions = d
		}
	}
	neighbors, err := p.neighbors(ctx)
	if err != nil {
		return nil, err
	}
	return portDetails(info, p.host, ports, descriptions, neighbors), nil
}

func portDetails(info SystemInfo, host string, ports []Port, descriptions map[string]string, neighbors []Neighbor) []domain.Detail {
	byPort := map[string]Neighbor{}
	for _, n := range neighbors {
		byPort[n.LocalPort] = n
	}
	up := 0
	for _, port := range ports {
		if portState(port) == "up" {
			up++
		}
	}
	details := []domain.Detail{
		{Label: "Hostname", Value: info.Hostname},
		{Label: "Model", Value: strings.ToUpper(info.Model)},
		{Label: "Junos", Value: info.Version},
		{Label: "Address", Value: host},
		{Label: "Ports up", Value: fmt.Sprintf("%d of %d", up, len(ports))},
	}
	for _, port := range ports {
		parts := []string{portState(port)}
		if d := descriptions[port.Name]; d != "" {
			parts = append(parts, d)
		}
		if n, ok := byPort[port.Name]; ok && n.System != "" {
			neighbor := n.System
			if n.RemotePort != "" {
				neighbor += " (" + n.RemotePort + ")"
			}
			parts = append(parts, "→ "+neighbor)
		}
		details = append(details, domain.Detail{Label: port.Name, Value: strings.Join(parts, " · ")})
	}
	return details
}

func portState(p Port) string {
	switch {
	case p.Admin == "down":
		return "off"
	case p.Oper == "up":
		return "up"
	}
	return "empty"
}

func kindFor(model string) domain.DeviceKind {
	m := strings.ToLower(model)
	switch {
	case strings.HasPrefix(m, "srx"):
		return domain.DeviceFirewall
	case strings.HasPrefix(m, "ex"), strings.HasPrefix(m, "qfx"):
		return domain.DeviceSwitch
	case strings.HasPrefix(m, "mx"), strings.HasPrefix(m, "acx"), strings.HasPrefix(m, "ptx"):
		return domain.DeviceRouter
	}
	return domain.DeviceAppliance
}
