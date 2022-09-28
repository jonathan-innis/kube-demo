package components

import (
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

func UtilizationBar(title string, used, total int64, opts ...progress.Option) string {
	opts = append(opts, progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		title,
		"  ",
		progress.New(opts...).ViewAs(float64(used)/float64(total)),
	)
}
