package proxmox

import (
	"context"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func (p *Provider) Status(ctx context.Context) (map[string]domain.Status, error) {
	nodes, err := p.api.nodes(ctx)
	if err != nil {
		return nil, err
	}
	guests, err := p.api.guests(ctx)
	if err != nil {
		return nil, err
	}
	statuses := make(map[string]domain.Status, len(nodes)+len(guests))
	for _, n := range nodes {
		statuses[p.nodeID(n.Node)] = nodeStatus(n.Status)
	}
	for _, g := range guests {
		statuses[p.guestID(g.VMID)] = guestStatus(g.Status)
	}
	return statuses, nil
}
