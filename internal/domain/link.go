package domain

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
