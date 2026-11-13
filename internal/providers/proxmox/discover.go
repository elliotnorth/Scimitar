package proxmox

import (
	"context"
	"sync"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

// addAddresses fills in the LAN address of each running guest. The lookups
// run side by side under one deadline, so guests whose agent is slow to
// answer cannot hold up discovery.
func (p *Provider) addAddresses(ctx context.Context, d *domain.Discovery, guests []apiGuest, online map[string]bool) {
	ctx, cancel := context.WithTimeout(ctx, addressBudget)
	defer cancel()
	byID := map[string]*domain.Device{}
	for i := range d.Devices {
		byID[d.Devices[i].ID] = &d.Devices[i]
	}
	var wg sync.WaitGroup
	slots := make(chan struct{}, addressLookups)
	for _, g := range guests {
		device := byID[p.guestID(g.VMID)]
		if device == nil || !online[g.Node] || g.Status != "running" {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			if address := p.api.guestAddress(ctx, g); address != "" {
				device.Addresses = []string{address}
			}
		}()
	}
	wg.Wait()
}

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
		device := domain.Device{
			ID:       id,
			Name:     g.Name,
			Kind:     guestKind(g.Type),
			Provider: p.name,
			Status:   guestStatus(g.Status),
		}
		d.Devices = append(d.Devices, device)
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
	p.addAddresses(ctx, &d, guests, online)
	return d, nil
}
