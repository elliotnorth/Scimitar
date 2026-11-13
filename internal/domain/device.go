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

	DeviceAccessPoint DeviceKind = "access-point"
	DeviceModem       DeviceKind = "modem"
	DeviceNAS         DeviceKind = "nas"
	DeviceServer      DeviceKind = "server"
	DeviceComputer    DeviceKind = "computer"
	DeviceSBC         DeviceKind = "single-board"
	DevicePrinter     DeviceKind = "printer"
	DeviceCamera      DeviceKind = "camera"
	DeviceNVR         DeviceKind = "nvr"
	DeviceUPS         DeviceKind = "ups"
	DevicePDU         DeviceKind = "pdu"
	DeviceIoT         DeviceKind = "iot"
	DeviceHub         DeviceKind = "smart-hub"
	DeviceMedia       DeviceKind = "media-player"
	DeviceTV          DeviceKind = "tv"
	DeviceConsole     DeviceKind = "console"
	DevicePhone       DeviceKind = "phone"
	DeviceKVM         DeviceKind = "kvm"
	DeviceOther       DeviceKind = "other"
)

// ManualKinds are the kinds a user can pick when adding a device by hand.
var ManualKinds = []DeviceKind{
	DeviceRouter, DeviceFirewall, DeviceSwitch, DeviceAccessPoint, DeviceModem,
	DeviceNAS, DeviceServer, DeviceHost, DeviceComputer, DeviceSBC,
	DevicePrinter, DeviceCamera, DeviceNVR, DeviceUPS, DevicePDU,
	DeviceIoT, DeviceHub, DeviceMedia, DeviceTV, DeviceConsole,
	DevicePhone, DeviceKVM, DeviceAppliance, DeviceOther,
}

type Device struct {
	ID        string
	Name      string
	Kind      DeviceKind
	Provider  string
	Addresses []string
	Status    Status
}
