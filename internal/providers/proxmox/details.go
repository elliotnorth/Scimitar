package proxmox

import (
	"context"
	"fmt"
	"strconv"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func (p *Provider) Details(ctx context.Context, deviceID string) ([]domain.Detail, error) {
	ref, err := p.parseID(deviceID)
	if err != nil {
		return nil, err
	}
	if ref.node != "" {
		return p.nodeDetails(ctx, ref.node)
	}
	return p.guestDetails(ctx, ref.vmid)
}

func (p *Provider) nodeDetails(ctx context.Context, name string) ([]domain.Detail, error) {
	nodes, err := p.api.nodes(ctx)
	if err != nil {
		return nil, err
	}
	for _, n := range nodes {
		if n.Node == name {
			return []domain.Detail{
				{Label: "Status", Value: n.Status},
				{Label: "CPUs", Value: strconv.Itoa(n.MaxCPU)},
				{Label: "Memory", Value: formatBytes(n.MaxMem)},
				{Label: "Uptime", Value: formatUptime(n.Uptime)},
			}, nil
		}
	}
	return nil, fmt.Errorf("proxmox: node %q not found", name)
}

func (p *Provider) guestDetails(ctx context.Context, vmid int) ([]domain.Detail, error) {
	guests, err := p.api.guests(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range guests {
		if g.VMID != vmid {
			continue
		}
		details := []domain.Detail{
			{Label: "VMID", Value: strconv.Itoa(g.VMID)},
			{Label: "Type", Value: g.Type},
			{Label: "Node", Value: g.Node},
			{Label: "Status", Value: g.Status},
			{Label: "CPUs", Value: strconv.Itoa(g.MaxCPU)},
			{Label: "Memory", Value: formatBytes(g.MaxMem)},
			{Label: "Uptime", Value: formatUptime(g.Uptime)},
		}
		if cfg, err := p.api.guestConfig(ctx, g); err == nil && cfg.Description != "" {
			details = append(details, domain.Detail{Label: "Notes", Value: cfg.Description})
		}
		return details, nil
	}
	return nil, fmt.Errorf("proxmox: guest %d not found", vmid)
}
