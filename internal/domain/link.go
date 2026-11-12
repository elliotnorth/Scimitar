package domain

import "strings"

type LinkKind string

const (
	LinkRunsOn    LinkKind = "runs-on"
	LinkConnected LinkKind = "connected"
)

type Link struct {
	From string
	To   string
	Kind LinkKind
}

const nameRefPrefix = "name:"

// NameRef points a link at a device by name when the provider does not know its ID,
// for example an LLDP neighbor found by another source. Core resolves it.
func NameRef(name string) string {
	return nameRefPrefix + name
}

// ParseNameRef returns the device name a link target refers to, if it is a name reference.
func ParseNameRef(to string) (string, bool) {
	return strings.CutPrefix(to, nameRefPrefix)
}
