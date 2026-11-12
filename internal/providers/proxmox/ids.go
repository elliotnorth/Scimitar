package proxmox

import (
	"fmt"
	"strconv"
	"strings"
)

func (p *Provider) nodeID(node string) string {
	return p.name + "/node/" + node
}

func (p *Provider) guestID(vmid int) string {
	return p.name + "/guest/" + strconv.Itoa(vmid)
}

type deviceRef struct {
	node string
	vmid int
}

func (p *Provider) parseID(id string) (deviceRef, error) {
	rest, ok := strings.CutPrefix(id, p.name+"/")
	if !ok {
		return deviceRef{}, fmt.Errorf("proxmox: device %q does not belong to %s", id, p.name)
	}
	if node, ok := strings.CutPrefix(rest, "node/"); ok && node != "" {
		return deviceRef{node: node}, nil
	}
	if raw, ok := strings.CutPrefix(rest, "guest/"); ok {
		if vmid, err := strconv.Atoi(raw); err == nil {
			return deviceRef{vmid: vmid}, nil
		}
	}
	return deviceRef{}, fmt.Errorf("proxmox: malformed device id %q", id)
}
