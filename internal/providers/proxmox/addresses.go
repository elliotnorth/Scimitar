package proxmox

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// Looking up guest addresses must never hold up discovery for long.
const (
	addressTimeout = 3 * time.Second
	addressBudget  = 6 * time.Second
	addressLookups = 8
)

type apiLXCInterface struct {
	Name string `json:"name"`
	Inet string `json:"inet"`
}

type apiAgentInterfaces struct {
	Result []struct {
		Name string `json:"name"`
		IPs  []struct {
			Address string `json:"ip-address"`
			Type    string `json:"ip-address-type"`
		} `json:"ip-addresses"`
	} `json:"result"`
}

// Interfaces that never carry the address other machines reach the guest at.
var internalInterfaces = []string{"lo", "docker", "br-", "veth", "cni", "flannel", "cali", "tailscale", "wg", "virbr", "podman"}

func usableAddress(iface, address string) string {
	for _, prefix := range internalInterfaces {
		if strings.HasPrefix(iface, prefix) {
			return ""
		}
	}
	address, _, _ = strings.Cut(address, "/")
	ip := net.ParseIP(address)
	if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return ""
	}
	return ip.String()
}

// guestAddress returns the LAN address of a running guest, or "" when Proxmox
// cannot tell: a VM without the guest agent, or a token that may not ask.
// Scimitar uses it to check the apps on the guest.
func (c *client) guestAddress(ctx context.Context, g apiGuest) string {
	if g.Status != "running" {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, addressTimeout)
	defer cancel()
	if g.Type == "lxc" {
		ifaces, err := get[[]apiLXCInterface](ctx, c, fmt.Sprintf("/nodes/%s/lxc/%d/interfaces", g.Node, g.VMID))
		if err != nil {
			return ""
		}
		for _, iface := range ifaces {
			if address := usableAddress(iface.Name, iface.Inet); address != "" {
				return address
			}
		}
		return ""
	}
	agent, err := get[apiAgentInterfaces](ctx, c, fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", g.Node, g.VMID))
	if err != nil {
		return ""
	}
	for _, iface := range agent.Result {
		for _, ip := range iface.IPs {
			if address := usableAddress(iface.Name, ip.Address); address != "" {
				return address
			}
		}
	}
	return ""
}
