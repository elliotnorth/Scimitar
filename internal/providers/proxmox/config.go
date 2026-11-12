package proxmox

import "errors"

type Config struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	TokenID     string `json:"token_id"`
	TokenSecret string `json:"token_secret"`
	Fingerprint string `json:"fingerprint"`
	CAFile      string `json:"ca_file"`
}

func (c Config) validate() error {
	switch {
	case c.URL == "":
		return errors.New("proxmox: url is required")
	case c.TokenID == "" || c.TokenSecret == "":
		return errors.New("proxmox: token id and secret are required")
	case c.Fingerprint != "" && c.CAFile != "":
		return errors.New("proxmox: set either fingerprint or ca file, not both")
	}
	return nil
}
