package views

type ViewType string

const (
	NodeView       ViewType = "Node"
	PodView        ViewType = "Pod"
	NodeDetailView ViewType = "NodeDetail"
	PodDetailView  ViewType = "PodDetail"
)
