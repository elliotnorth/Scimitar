package junos

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

const defaultSSHPort = "22"

// Config describes one Juniper device. Each device is its own provider, named after it.
type Config struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	Username string `json:"username"`
	HostKey  string `json:"host_key"`
	KeyFile  string `json:"key_file,omitempty"`
}

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$`)

// ValidName reports whether name can be used as a device name. Names become part of device IDs.
func ValidName(name string) bool {
	return validName.MatchString(name)
}

func (c Config) validate() error {
	switch {
	case !ValidName(c.Name):
		return fmt.Errorf("junos: name %q must be letters, numbers, dots, dashes or underscores", c.Name)
	case c.Address == "":
		return errors.New("junos: address is required")
	case c.Username == "":
		return errors.New("junos: username is required")
	case !strings.HasPrefix(c.HostKey, "SHA256:"):
		return errors.New("junos: host_key must be a SHA256 fingerprint, like SHA256:abc...")
	case c.KeyFile == "":
		return errors.New("junos: key_file is required")
	}
	return nil
}

// NormalizeAddress adds the SSH port to an address that has none.
func NormalizeAddress(address string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", errors.New("address is empty")
	}
	if host, port, err := net.SplitHostPort(address); err == nil {
		if host == "" || port == "" {
			return "", fmt.Errorf("%q is not a valid address", address)
		}
		return address, nil
	}
	host := strings.TrimSuffix(strings.TrimPrefix(address, "["), "]")
	if strings.ContainsAny(host, " /") {
		return "", fmt.Errorf("%q is not a valid address", address)
	}
	return net.JoinHostPort(host, defaultSSHPort), nil
}
