package core

import "github.com/elliotnorth/Scimitar/internal/domain"

func (inv *Inventory) rebuild() {
	next := Snapshot{Errors: map[string]string{}, UpdatedAt: inv.now()}
	var hints []domain.ServiceHint
	deviceStatus := map[string]domain.Status{}
	deviceAddress := map[string]string{}

	for _, p := range inv.registry.Providers() {
		next.Providers = append(next.Providers, p.Name())
		state, ok := inv.states[p.Name()]
		if !ok {
			continue
		}
		if err := state.err(); err != nil {
			next.Errors[p.Name()] = err.Error()
		}
		for _, d := range state.discovery.Devices {
			d.Status = currentStatus(d, state)
			deviceStatus[d.ID] = d.Status
			if len(d.Addresses) > 0 {
				deviceAddress[d.ID] = d.Addresses[0]
			}
			next.Devices = append(next.Devices, d)
		}
		next.Links = append(next.Links, state.discovery.Links...)
		hints = append(hints, state.discovery.Hints...)
	}
	next.Links = resolveLinks(next.Links, next.Devices)

	matched := Match(inv.catalog, hints)
	next.Answered = matched.applyDecisions(inv.catalog, inv.decisions)
	for i := range matched.Services {
		svc := &matched.Services[i]
		svc.URL, svc.URLSet = inv.appURL(*svc, deviceAddress[svc.DeviceID])
		svc.Status = inv.serviceStatus(deviceStatus[svc.DeviceID], svc.URL)
	}
	next.Services = matched.Services
	next.Review = matched.Review
	next.Events = recentEvents(inv.events)
	inv.snapshot = next
}

func currentStatus(d domain.Device, state *providerState) domain.Status {
	if state.err() != nil {
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

// serviceStatus is down when the device is down, otherwise whatever the last
// check of the service's address found. A service nobody can check stays
// unknown, so a running VM never makes it look up by guesswork.
func (inv *Inventory) serviceStatus(device domain.Status, url string) domain.Status {
	if device == domain.StatusDown {
		return domain.StatusDown
	}
	if status, checked := inv.appStatus[url]; checked && url != "" {
		return status
	}
	return domain.StatusUnknown
}
