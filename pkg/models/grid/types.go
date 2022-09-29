package grid

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewOverride func(lipgloss.Style) lipgloss.Style

type Interface[T any, U tea.Msg] interface {
	InitFromMsg(U) T
	Update(tea.Msg) (T, tea.Cmd)
	View(...ViewOverride) string
	GetViewportContent() string
	GetCreationTimestamp() int64
	GetUID() string
}

type MessageInterface interface {
	GetID() string
}
