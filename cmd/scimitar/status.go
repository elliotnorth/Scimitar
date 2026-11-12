package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
)

func printStatus(out io.Writer, configPath string, paths installPaths) {
	fmt.Fprintf(out, "Scimitar %s\n\n", version)
	if !fileExists(paths.unit) {
		fmt.Fprintln(out, "  Service:   not installed. Run: sudo scimitar install")
		return
	}
	fmt.Fprintf(out, "  Service:   %s\n", serviceState())

	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintln(out, "  Settings:  run with sudo to see them")
		return
	}
	host, port, _ := net.SplitHostPort(cfg.Listen)
	if ip := primaryIP(); host == "0.0.0.0" && ip != "" {
		host = ip
	}
	fmt.Fprintf(out, "  Open:      https://%s\n", net.JoinHostPort(host, port))
	for _, p := range cfg.Providers {
		var settings struct {
			URL string `json:"url"`
		}
		json.Unmarshal(p.Settings, &settings)
		fmt.Fprintf(out, "  Source:    %s at %s\n", p.Type, settings.URL)
	}
	fmt.Fprintf(out, "  Data:      %s\n", cfg.DataDir)
	fmt.Fprintln(out, "\n  Logs:      journalctl -u scimitar -f")
}

func serviceState() string {
	out, _ := exec.Command("systemctl", "is-active", "scimitar").Output()
	switch state := strings.TrimSpace(string(out)); state {
	case "active":
		return "running"
	case "":
		return "unknown"
	default:
		return state + ". Check: journalctl -u scimitar -n 30"
	}
}
