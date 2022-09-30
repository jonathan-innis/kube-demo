package views

import tea "github.com/charmbracelet/bubbletea"

type Type string

const (
	NodeType       Type = "Node"
	PodType        Type = "Pod"
	NodeDetailType Type = "NodeDetail"
	PodDetailType  Type = "PodDetail"
)

type Mode string

const (
	ViewMode        Mode = "ViewMode"
	InteractiveMode Mode = "InteractiveMode"
)

type ViewTypeChangeMsg struct {
	ActiveView Type
}

func ChangeViewType(vt Type) tea.Cmd {
	return func() tea.Msg { return ViewTypeChangeMsg{ActiveView: vt} }
}

type ViewModeChangeMsg struct {
	ActiveMode Mode
}

func ChangeViewMode(mode Mode) tea.Cmd {
	return func() tea.Msg { return ViewModeChangeMsg{ActiveMode: mode} }
}