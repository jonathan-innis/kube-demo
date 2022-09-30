package style

import "github.com/charmbracelet/lipgloss"

var Canvas = lipgloss.NewStyle().Padding(1, 2, 1, 2)

var Wrapper = lipgloss.NewStyle().Margin(0).Padding(0)

var Node = lipgloss.NewStyle().
	Align(lipgloss.Left).
	Foreground(White).
	Border(lipgloss.HiddenBorder(), true).
	BorderBackground(nodeBorder).
	Margin(1).
	Padding(0, 1, 0, 1).
	Height(10).
	Width(30)

var Pod = lipgloss.NewStyle().
	Align(lipgloss.Bottom).
	Foreground(White).
	Border(lipgloss.RoundedBorder(), true).
	BorderForeground(defaultPodBorder).
	Margin(0).
	Padding(0).
	Height(0).
	Width(1)

const (
	White  = lipgloss.Color("#FFFFFF")
	Black  = lipgloss.Color("#000000")
	Orange = lipgloss.Color("#FFA500")
	Pink   = lipgloss.Color("#F87575")
	Teal   = lipgloss.Color("#27CEBD")
	Grey   = lipgloss.Color("#6C7D89")
	Yellow = lipgloss.Color("#FFFF00")
	Red    = lipgloss.Color("#FF0000")

	SelectedBorder = Pink

	Separator = "-----------"
)

var nodeBorder = Grey
var defaultPodBorder = Teal
