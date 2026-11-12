package proxmox

import (
	"net/http"
	"time"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

const requestTimeout = 10 * time.Second

var _ domain.Provider = (*Provider)(nil)

type Provider struct {
	name string
	api  *client
}

func New(cfg Config) (*Provider, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	tlsCfg, err := tlsConfig(cfg)
	if err != nil {
		return nil, err
	}
	name := cfg.Name
	if name == "" {
		name = "proxmox"
	}
	return &Provider{
		name: name,
		api: &client{
			baseURL: trimBaseURL(cfg.URL),
			auth:    tokenHeader(cfg.TokenID, cfg.TokenSecret),
			http: &http.Client{
				Timeout:   requestTimeout,
				Transport: &http.Transport{TLSClientConfig: tlsCfg},
			},
		},
	}, nil
}

func (p *Provider) Name() string {
	return p.name
}
