package domain

type Service struct {
	ID        string
	Name      string
	CatalogID string
	DeviceID  string
	Status    Status
	// URL is where the service is checked and opened. URLSet tells a URL the
	// user typed apart from one worked out from the device address.
	URL    string
	URLSet bool
}

type ServiceHint struct {
	DeviceID string
	Source   string
	Value    string
}
