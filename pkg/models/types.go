package models

type ViewType string

const (
	NodeView       ViewType = "Node"
	PodView        ViewType = "Pod"
	NodeDetailView ViewType = "NodeDetail"
	PodDetailView  ViewType = "PodDetail"
)

type Mode string

const (
	View        Mode = "View"
	Interactive Mode = "Interactive"
)
