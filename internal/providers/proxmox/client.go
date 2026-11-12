package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type client struct {
	baseURL string
	auth    string
	http    *http.Client
}

func get[T any](ctx context.Context, c *client, path string) (T, error) {
	var envelope struct{ Data T }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api2/json"+path, nil)
	if err != nil {
		return envelope.Data, err
	}
	req.Header.Set("Authorization", c.auth)
	resp, err := c.http.Do(req)
	if err != nil {
		return envelope.Data, fmt.Errorf("proxmox: GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return envelope.Data, fmt.Errorf("proxmox: GET %s: %s", path, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return envelope.Data, fmt.Errorf("proxmox: decode %s: %w", path, err)
	}
	return envelope.Data, nil
}

func tokenHeader(tokenID, secret string) string {
	return "PVEAPIToken=" + tokenID + "=" + secret
}

func trimBaseURL(url string) string {
	return strings.TrimSuffix(url, "/")
}
