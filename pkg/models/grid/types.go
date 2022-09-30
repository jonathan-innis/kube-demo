package grid

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewType string

const (
	Standard ViewType = "Standard"
	Detail   ViewType = "Detail"
	Single   ViewType = "Single"
)

type ViewOverride func(lipgloss.Style) lipgloss.Style

type Interface[T any, U tea.Msg] interface {
	Update(tea.Msg) (T, tea.Cmd)
	View(ViewType, ...ViewOverride) string
	GetViewportContent() string
	GetCreationTimestamp() int64
	GetUID() string
}

type MessageInterface interface {
	GetID() string
}
