package core

import "github.com/elliotnorth/Scimitar/internal/domain"

func (inv *Inventory) rebuild() {
	next := Snapshot{Errors: map[string]string{}, UpdatedAt: inv.now()}
	var hints []domain.ServiceHint
	deviceStatus := map[string]domain.Status{}

	for _, p := range inv.registry.Providers() {
		next.Providers = append(next.Providers, p.Name())
		state, ok := inv.states[p.Name()]
		if !ok {
			continue
		}
		if state.err != nil {
			next.Errors[p.Name()] = state.err.Error()
		}
		for _, d := range state.discovery.Devices {
			d.Status = currentStatus(d, state)
			deviceStatus[d.ID] = d.Status
			next.Devices = append(next.Devices, d)
		}
		next.Links = append(next.Links, state.discovery.Links...)
		hints = append(hints, state.discovery.Hints...)
	}

	matched := Match(inv.catalog, hints)
	matched.applyDecisions(inv.catalog, inv.decisions)
	for i := range matched.Services {
		matched.Services[i].Status = serviceStatus(deviceStatus[matched.Services[i].DeviceID])
	}
	next.Services = matched.Services
	next.Review = matched.Review
	inv.snapshot = next
}

func currentStatus(d domain.Device, state *providerState) domain.Status {
	if state.err != nil {
		return domain.StatusUnknown
	}
	if s, ok := state.status[d.ID]; ok {
		return s
	}
	if state.status != nil {
		return domain.StatusUnknown
	}
	return d.Status
}

func serviceStatus(device domain.Status) domain.Status {
	if device == domain.StatusDown {
		return domain.StatusDown
	}
	return domain.StatusUnknown
}
