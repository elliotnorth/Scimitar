// Package alerts sends a message to the user's chat or phone when something
// goes down or comes back.
package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/elliotnorth/Scimitar/internal/core"
	"github.com/elliotnorth/Scimitar/internal/domain"
	"github.com/elliotnorth/Scimitar/internal/jsonfile"
)

const (
	maxChannels = 10
	sendTimeout = 10 * time.Second
)

// Channel types. Each is a plain HTTPS request to an address the user pastes in.
const (
	TypeNtfy    = "ntfy"
	TypeDiscord = "discord"
	TypeSlack   = "slack"
	TypeGotify  = "gotify"
	TypeWebhook = "webhook"
)

var ErrInvalid = errors.New("invalid alert settings")

type Channel struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Settings struct {
	Channels []Channel `json:"channels"`
	// Devices, Apps and Sources choose what to be told about.
	Devices bool `json:"devices"`
	Apps    bool `json:"apps"`
	Sources bool `json:"sources"`
}

func defaults() Settings {
	return Settings{Channels: []Channel{}, Devices: true, Apps: true, Sources: true}
}

// Result is the outcome of sending to one channel.
type Result struct {
	Type  string `json:"type"`
	Error string `json:"error,omitempty"`
}

type Notifier struct {
	path   string
	client *http.Client

	mu       sync.Mutex
	settings Settings
}

func Open(path string) (*Notifier, error) {
	n := &Notifier{path: path, client: &http.Client{Timeout: sendTimeout}, settings: defaults()}
	if err := jsonfile.Load(path, &n.settings); err != nil {
		return nil, fmt.Errorf("load alert settings: %w", err)
	}
	if n.settings.Channels == nil {
		n.settings.Channels = []Channel{}
	}
	return n, nil
}

func (n *Notifier) Settings() Settings {
	n.mu.Lock()
	defer n.mu.Unlock()
	s := n.settings
	s.Channels = append([]Channel{}, s.Channels...)
	return s
}

func validate(s Settings) (Settings, error) {
	if len(s.Channels) > maxChannels {
		return s, fmt.Errorf("%w: at most %d places to send alerts", ErrInvalid, maxChannels)
	}
	out := s
	out.Channels = make([]Channel, 0, len(s.Channels))
	for _, c := range s.Channels {
		c.URL = strings.TrimSpace(c.URL)
		switch c.Type {
		case TypeNtfy, TypeDiscord, TypeSlack, TypeGotify, TypeWebhook:
		default:
			return s, fmt.Errorf("%w: unknown alert type %q", ErrInvalid, c.Type)
		}
		u, err := url.Parse(c.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || len(c.URL) > 1024 {
			return s, fmt.Errorf("%w: the %s address must start with https:// or http://", ErrInvalid, c.Type)
		}
		out.Channels = append(out.Channels, c)
	}
	return out, nil
}

// Update checks and saves new settings.
func (n *Notifier) Update(s Settings) error {
	s, err := validate(s)
	if err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if err := jsonfile.Save(n.path, s); err != nil {
		return fmt.Errorf("save alert settings: %w", err)
	}
	n.settings = s
	return nil
}

type message struct {
	Title string
	Body  string
	Down  bool
	Event core.Event
}

func describe(e core.Event) message {
	m := message{Down: e.Status == domain.StatusDown, Event: e, Body: e.Detail}
	switch {
	case e.Subject == core.SubjectSource && m.Down:
		m.Title = "Scimitar can't reach " + e.Name
	case e.Subject == core.SubjectSource:
		m.Title = "Scimitar can reach " + e.Name + " again"
	case m.Down:
		m.Title = e.Name + " is down"
	default:
		m.Title = e.Name + " is back up"
	}
	return m
}

func (s Settings) wants(e core.Event) bool {
	switch e.Subject {
	case core.SubjectDevice:
		return s.Devices
	case core.SubjectApp:
		return s.Apps
	case core.SubjectSource:
		return s.Sources
	}
	return false
}

// Notify sends each event to every channel in the background.
func (n *Notifier) Notify(events []core.Event) {
	s := n.Settings()
	if len(s.Channels) == 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*sendTimeout)
		defer cancel()
		for _, e := range events {
			if !s.wants(e) {
				continue
			}
			for _, r := range n.sendAll(ctx, s.Channels, describe(e)) {
				if r.Error != "" {
					slog.Warn("alert was not delivered", "type", r.Type, "err", r.Error)
				}
			}
		}
	}()
}

// Test sends a sample message to every channel and reports how each one went.
func (n *Notifier) Test(ctx context.Context) []Result {
	return n.sendAll(ctx, n.Settings().Channels, message{
		Title: "Scimitar test alert",
		Body:  "If you can read this, alerts are working.",
		Event: core.Event{Time: time.Now(), Subject: "test", Name: "test", Status: domain.StatusUp},
	})
}

func (n *Notifier) sendAll(ctx context.Context, channels []Channel, m message) []Result {
	results := make([]Result, len(channels))
	var wg sync.WaitGroup
	for i, c := range channels {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = Result{Type: c.Type}
			if err := n.send(ctx, c, m); err != nil {
				results[i].Error = err.Error()
			}
		}()
	}
	wg.Wait()
	return results
}

func (n *Notifier) send(ctx context.Context, c Channel, m message) error {
	text := m.Title
	if m.Body != "" {
		text += "\n" + m.Body
	}
	var body []byte
	header := http.Header{"Content-Type": {"application/json"}}
	switch c.Type {
	case TypeNtfy:
		body = []byte(m.Body)
		if m.Body == "" {
			body = []byte(m.Title)
		}
		header = http.Header{"Content-Type": {"text/plain; charset=utf-8"}, "Title": {m.Title}, "Tags": {"white_check_mark"}}
		if m.Down {
			header.Set("Priority", "high")
			header.Set("Tags", "rotating_light")
		}
	case TypeDiscord:
		body, _ = json.Marshal(map[string]string{"content": text})
	case TypeSlack:
		body, _ = json.Marshal(map[string]string{"text": text})
	case TypeGotify:
		priority := 4
		if m.Down {
			priority = 8
		}
		body, _ = json.Marshal(map[string]any{"title": m.Title, "message": text, "priority": priority})
	default:
		body, _ = json.Marshal(map[string]any{
			"title": m.Title, "detail": m.Body, "subject": m.Event.Subject, "id": m.Event.ID,
			"name": m.Event.Name, "status": m.Event.Status, "time": m.Event.Time.UTC().Format(time.RFC3339),
		})
	}
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return errors.New("the address is not valid")
	}
	req.Header = header
	req.Header.Set("User-Agent", "scimitar")
	resp, err := n.client.Do(req)
	if err != nil {
		// The address can hold a secret token, so it is kept out of the error.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return fmt.Errorf("could not send: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("the service answered %s", resp.Status)
	}
	return nil
}
