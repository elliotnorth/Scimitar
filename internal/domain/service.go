package domain

type Service struct {
	ID        string
	Name      string
	CatalogID string
	DeviceID  string
	Status    Status
}

type ServiceHint struct {
	DeviceID string
	Source   string
	Value    string
}
