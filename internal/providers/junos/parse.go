package junos

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// Junos "| display xml" output is read into a tree where every element is a
// map from child names to lists of children, and an element's text is stored
// under "data". Results nest differently for standalone devices, virtual
// chassis and clusters, so these helpers search by name and handle all three.

type SystemInfo struct {
	Hostname string
	Model    string
	Version  string
}

type Port struct {
	Name        string
	Admin       string
	Oper        string
	Description string
}

type Neighbor struct {
	LocalPort  string
	System     string
	RemotePort string
}

var physicalPort = regexp.MustCompile(`^(ge|xe|et|mge|fe)-\d+/\d+/\d+$`)

func decode(out []byte) (any, error) {
	unexpected := fmt.Errorf("junos: unexpected output: %s", firstLine(string(out)))
	root := map[string]any{}
	stack := []map[string]any{root}
	var texts []*strings.Builder
	dec := xml.NewDecoder(bytes.NewReader(out))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, unexpected
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child := map[string]any{}
			parent := stack[len(stack)-1]
			list, _ := parent[t.Name.Local].([]any)
			parent[t.Name.Local] = append(list, child)
			stack = append(stack, child)
			texts = append(texts, &strings.Builder{})
		case xml.CharData:
			if len(texts) > 0 {
				texts[len(texts)-1].Write(t)
			}
		case xml.EndElement:
			if len(texts) == 0 {
				return nil, unexpected
			}
			if s := strings.TrimSpace(texts[len(texts)-1].String()); s != "" {
				stack[len(stack)-1]["data"] = s
			}
			stack = stack[:len(stack)-1]
			texts = texts[:len(texts)-1]
		}
	}
	if len(root) == 0 || len(stack) != 1 {
		return nil, unexpected
	}
	return root, nil
}

// find returns the first value stored under key, searching depth first in a fixed order.
func find(v any, key string) any {
	switch t := v.(type) {
	case map[string]any:
		if val, ok := t[key]; ok {
			return val
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if r := find(t[k], key); r != nil {
				return r
			}
		}
	case []any:
		for _, item := range t {
			if r := find(item, key); r != nil {
				return r
			}
		}
	}
	return nil
}

func objects(v any) []map[string]any {
	items, _ := v.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func text(obj map[string]any, keys ...string) string {
	for _, key := range keys {
		items := objects(obj[key])
		if len(items) == 0 {
			continue
		}
		if s, ok := items[0]["data"].(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				return s
			}
		}
	}
	return ""
}

func parseVersion(out []byte) (SystemInfo, error) {
	v, err := decode(out)
	if err != nil {
		return SystemInfo{}, err
	}
	sw := objects(find(v, "software-information"))
	if len(sw) == 0 {
		return SystemInfo{}, errors.New("junos: show version returned no software information")
	}
	info := SystemInfo{
		Hostname: text(sw[0], "host-name"),
		Model:    text(sw[0], "product-model", "product-name"),
		Version:  text(sw[0], "junos-version"),
	}
	if info.Version == "" {
		info.Version = packageVersion(sw[0])
	}
	if info.Model == "" {
		return SystemInfo{}, errors.New("junos: show version returned no model")
	}
	return info, nil
}

// packageVersion reads the version from older releases, which only list it as "JUNOS ... [15.1X53-D59.4]".
func packageVersion(sw map[string]any) string {
	for _, pkg := range objects(sw["package-information"]) {
		comment := text(pkg, "comment")
		if start, end := strings.Index(comment, "["), strings.LastIndex(comment, "]"); start >= 0 && end > start {
			return comment[start+1 : end]
		}
	}
	return ""
}

func parsePorts(out []byte) ([]Port, error) {
	v, err := decode(out)
	if err != nil {
		return nil, err
	}
	var ports []Port
	for _, iface := range objects(find(v, "physical-interface")) {
		name := text(iface, "name")
		if !physicalPort.MatchString(name) {
			continue
		}
		ports = append(ports, Port{Name: name, Admin: text(iface, "admin-status"), Oper: text(iface, "oper-status")})
	}
	return ports, nil
}

func parseDescriptions(out []byte) (map[string]string, error) {
	descriptions := map[string]string{}
	if len(bytes.TrimSpace(out)) == 0 {
		return descriptions, nil
	}
	v, err := decode(out)
	if err != nil {
		return nil, err
	}
	for _, iface := range objects(find(v, "physical-interface")) {
		if d := text(iface, "description"); d != "" {
			descriptions[text(iface, "name")] = d
		}
	}
	return descriptions, nil
}

func parseNeighbors(out []byte) ([]Neighbor, error) {
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, nil
	}
	v, err := decode(out)
	if err != nil {
		return nil, err
	}
	var neighbors []Neighbor
	for _, n := range objects(find(v, "lldp-neighbor-information")) {
		neighbors = append(neighbors, Neighbor{
			LocalPort:  text(n, "lldp-local-port-id", "lldp-local-interface"),
			System:     text(n, "lldp-remote-system-name"),
			RemotePort: text(n, "lldp-remote-port-description", "lldp-remote-port-id"),
		})
	}
	return neighbors, nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
