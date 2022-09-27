package model

import "github.com/charmbracelet/bubbles/key"

type keyMap map[string]key.Binding

var keyMappings = keyMap{
	"Move": key.NewBinding(
		key.WithKeys("up", "down", "left", "right"),
		key.WithHelp("↑/↓/←/→", "move"),
	),
	"Help": key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	"Quit": key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
