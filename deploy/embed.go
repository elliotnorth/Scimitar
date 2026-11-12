package deploy

import _ "embed"

//go:embed scimitar.service
var SystemdUnit string
