package domain

import "context"

type Discovery struct {
	Devices []Device
	Links   []Link
	Hints   []ServiceHint
}

type Detail struct {
	Label string
	Value string
}

type Provider interface {
	Name() string
	Discover(ctx context.Context) (Discovery, error)
	Status(ctx context.Context) (map[string]Status, error)
	Details(ctx context.Context, deviceID string) ([]Detail, error)
}
