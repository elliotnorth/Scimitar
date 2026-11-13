package core

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/jsonfile"
)

const (
	maxStoredEvents = 300
	maxShownEvents  = 50
)

type EventSubject string

const (
	SubjectDevice EventSubject = "device"
	SubjectApp    EventSubject = "app"
	SubjectSource EventSubject = "source"
)

// Event records something going down or coming back.
type Event struct {
	Time    time.Time     `json:"time"`
	Subject EventSubject  `json:"subject"`
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Status  domain.Status `json:"status"`
	Detail  string        `json:"detail,omitempty"`
}

// EventStore keeps the event history across restarts.
type EventStore interface {
	Load() ([]Event, error)
	Save([]Event) error
}

type FileEvents struct {
	Path string
}

func (f FileEvents) Load() ([]Event, error) {
	var events []Event
	err := jsonfile.Load(f.Path, &events)
	return events, err
}

func (f FileEvents) Save(events []Event) error {
	return jsonfile.Save(f.Path, events)
}

// Extras are the optional parts of an inventory. Without them it still works,
// but apps are never checked and nothing is remembered or announced.
type Extras struct {
	Prober  Prober
	AppURLs AppURLStore
	Events  EventStore
	// Notify is called with each batch of new events. It must not block.
	Notify func([]Event)
}

// Configure sets the extras. Call it before the inventory starts running.
func (inv *Inventory) Configure(x Extras) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if x.AppURLs != nil {
		urls, err := x.AppURLs.Load()
		if err != nil {
			return fmt.Errorf("load app addresses: %w", err)
		}
		inv.appURLs = urls
	}
	if x.Events != nil {
		events, err := x.Events.Load()
		if err != nil {
			// History is nice to have; a damaged file should not stop Scimitar.
			slog.Warn("event history could not be read, starting fresh", "err", err)
			events = nil
		}
		inv.events = events
	}
	inv.prober, inv.urlStore, inv.eventStore, inv.notify = x.Prober, x.AppURLs, x.Events, x.Notify
	return nil
}

func (inv *Inventory) saveEvents(events []Event) {
	if inv.eventStore == nil {
		return
	}
	if err := inv.eventStore.Save(events); err != nil {
		slog.Warn("event history could not be saved", "err", err)
	}
}

// recentEvents returns the newest events first.
func recentEvents(events []Event) []Event {
	n := min(len(events), maxShownEvents)
	out := make([]Event, 0, n)
	for i := len(events) - 1; i >= len(events)-n; i-- {
		out = append(out, events[i])
	}
	return out
}

// detectChanges compares the snapshot with the last status seen for each
// device, app and source, and returns an event for every one that went down or
// came back. The first sighting of anything is never an event, and "unknown"
// is skipped over, so up, unknown, down still counts as going down.
func (inv *Inventory) detectChanges() []Event {
	s := inv.snapshot
	now := inv.now()
	var events []Event
	observe := func(subject EventSubject, id, name string, status domain.Status, detail string) {
		if status != domain.StatusUp && status != domain.StatusDown {
			return
		}
		key := string(subject) + ":" + id
		last, seen := inv.lastKnown[key]
		inv.lastKnown[key] = status
		if seen && last != status {
			events = append(events, Event{Time: now, Subject: subject, ID: id, Name: name, Status: status, Detail: detail})
		}
	}

	deviceName := map[string]string{}
	deviceDown := map[string]bool{}
	for _, d := range s.Devices {
		deviceName[d.ID] = d.Name
		deviceDown[d.ID] = d.Status == domain.StatusDown
		observe(SubjectDevice, d.ID, d.Name, d.Status, string(d.Kind)+" on "+d.Provider)
	}
	for _, svc := range s.Services {
		// An app on a device that is down is covered by the device's own event.
		if deviceDown[svc.DeviceID] {
			continue
		}
		observe(SubjectApp, svc.ID, svc.Name, svc.Status, "on "+deviceName[svc.DeviceID])
	}
	for _, name := range s.Providers {
		if _, tried := inv.states[name]; !tried {
			continue
		}
		status, detail := domain.StatusUp, ""
		if msg, failing := s.Errors[name]; failing {
			status, detail = domain.StatusDown, msg
		}
		observe(SubjectSource, name, name, status, detail)
	}
	return events
}
