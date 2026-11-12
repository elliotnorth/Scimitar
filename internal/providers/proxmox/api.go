package proxmox

import (
	"context"
	"fmt"
)

type apiNode struct {
	Node   string `json:"node"`
	Status string `json:"status"`
	MaxCPU int    `json:"maxcpu"`
	MaxMem int64  `json:"maxmem"`
	Uptime int64  `json:"uptime"`
}

type apiGuest struct {
	Type     string `json:"type"`
	Node     string `json:"node"`
	VMID     int    `json:"vmid"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Tags     string `json:"tags"`
	Template int    `json:"template"`
	MaxCPU   int    `json:"maxcpu"`
	MaxMem   int64  `json:"maxmem"`
	Uptime   int64  `json:"uptime"`
}

type apiGuestConfig struct {
	Description string `json:"description"`
}

func (c *client) nodes(ctx context.Context) ([]apiNode, error) {
	return get[[]apiNode](ctx, c, "/nodes")
}

func (c *client) guests(ctx context.Context) ([]apiGuest, error) {
	all, err := get[[]apiGuest](ctx, c, "/cluster/resources?type=vm")
	if err != nil {
		return nil, err
	}
	guests := all[:0]
	for _, g := range all {
		if g.Template == 0 {
			guests = append(guests, g)
		}
	}
	return guests, nil
}

func (c *client) guestConfig(ctx context.Context, g apiGuest) (apiGuestConfig, error) {
	return get[apiGuestConfig](ctx, c, fmt.Sprintf("/nodes/%s/%s/%d/config", g.Node, g.Type, g.VMID))
}
