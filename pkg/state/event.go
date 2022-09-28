package state

type EventKind string

const (
	NodeKind EventKind = "Node"
	PodKind  EventKind = "Pod"
)

type EventType string

const (
	Create EventType = "Create"
	Update EventType = "Update"
	Delete EventType = "Delete"
)

type Event struct {
	Kind EventKind
	Type EventType
	Obj  interface{}
}
