package models

import "github.com/charmbracelet/bubbles/key"

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k["Move"], k["Quit"], k["Help"], k["YAML"], k["JSON"]}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["Move"], k["Help"], k["Quit"], k["YAML"], k["JSON"]},
	}
}

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
	"YAML": key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "view yaml"),
	),
	"JSON": key.NewBinding(
		key.WithKeys("j"),
		key.WithHelp("j", "view json"),
	),
	"Quit": key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
