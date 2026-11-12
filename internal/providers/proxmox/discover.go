package proxmox

import (
	"context"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func (p *Provider) Discover(ctx context.Context) (domain.Discovery, error) {
	nodes, err := p.api.nodes(ctx)
	if err != nil {
		return domain.Discovery{}, err
	}
	guests, err := p.api.guests(ctx)
	if err != nil {
		return domain.Discovery{}, err
	}

	var d domain.Discovery
	online := map[string]bool{}
	for _, n := range nodes {
		online[n.Node] = n.Status == "online"
		d.Devices = append(d.Devices, domain.Device{
			ID:       p.nodeID(n.Node),
			Name:     n.Node,
			Kind:     domain.DeviceHost,
			Provider: p.name,
			Status:   nodeStatus(n.Status),
		})
	}
	for _, g := range guests {
		id := p.guestID(g.VMID)
		d.Devices = append(d.Devices, domain.Device{
			ID:       id,
			Name:     g.Name,
			Kind:     guestKind(g.Type),
			Provider: p.name,
			Status:   guestStatus(g.Status),
		})
		d.Links = append(d.Links, domain.Link{From: id, To: p.nodeID(g.Node), Kind: domain.LinkRunsOn})
		d.Hints = append(d.Hints, domain.ServiceHint{DeviceID: id, Source: "name", Value: g.Name})
		for _, tag := range splitTags(g.Tags) {
			d.Hints = append(d.Hints, domain.ServiceHint{DeviceID: id, Source: "tag", Value: tag})
		}
		if !online[g.Node] {
			continue
		}
		cfg, err := p.api.guestConfig(ctx, g)
		if err != nil {
			return domain.Discovery{}, err
		}
		if cfg.Description != "" {
			d.Hints = append(d.Hints, domain.ServiceHint{DeviceID: id, Source: "notes", Value: cfg.Description})
		}
	}
	return d, nil
}
