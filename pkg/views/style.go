package views

import "github.com/charmbracelet/lipgloss"

var canvasStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

var nodeStyle = lipgloss.NewStyle().
	Align(lipgloss.Left).
	Foreground(white).
	Border(lipgloss.HiddenBorder(), true).
	BorderBackground(nodeBorder).
	Margin(1).
	Padding(0, 1, 0, 1).
	Height(10).
	Width(30)

var podStyle = lipgloss.NewStyle().
	Align(lipgloss.Bottom).
	Foreground(white).
	Border(lipgloss.NormalBorder(), true).
	BorderForeground(defaultPodBorder).
	Margin(0).
	Padding(0).
	Height(0).
	Width(1)

const (
	white  = lipgloss.Color("#FFFFFF")
	black  = lipgloss.Color("#000000")
	orange = lipgloss.Color("#FFA500")
	pink   = lipgloss.Color("#F87575")
	teal   = lipgloss.Color("#27CEBD")
	grey   = lipgloss.Color("#6C7D89")
	yellow = lipgloss.Color("#FFFF00")
	red    = lipgloss.Color("#FF0000")
)

var nodeBorder = grey
var selectedNodeBorder = pink
var defaultPodBorder = teal
