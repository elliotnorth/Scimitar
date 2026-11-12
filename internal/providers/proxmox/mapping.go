package proxmox

import (
	"strings"

	"github.com/elliotnorth/Scimitar/internal/domain"
)

func nodeStatus(s string) domain.Status {
	switch s {
	case "online":
		return domain.StatusUp
	case "offline":
		return domain.StatusDown
	}
	return domain.StatusUnknown
}

func guestStatus(s string) domain.Status {
	switch s {
	case "running":
		return domain.StatusUp
	case "stopped":
		return domain.StatusDown
	}
	return domain.StatusUnknown
}

func guestKind(apiType string) domain.DeviceKind {
	if apiType == "lxc" {
		return domain.DeviceContainer
	}
	return domain.DeviceVM
}

func splitTags(tags string) []string {
	return strings.FieldsFunc(tags, func(r rune) bool {
		return r == ';' || r == ',' || r == ' '
	})
}
