package domain

type DeviceKind string

const (
	DeviceHost      DeviceKind = "host"
	DeviceVM        DeviceKind = "vm"
	DeviceContainer DeviceKind = "container"
	DeviceAppliance DeviceKind = "appliance"
	DeviceSwitch    DeviceKind = "switch"
	DeviceFirewall  DeviceKind = "firewall"
	DeviceRouter    DeviceKind = "router"
)

type Device struct {
	ID        string
	Name      string
	Kind      DeviceKind
	Provider  string
	Addresses []string
	Status    Status
}
