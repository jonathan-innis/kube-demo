package views

import "github.com/charmbracelet/lipgloss"

func GetBoxesPerRow(container lipgloss.Style, subContainer lipgloss.Style) int {
	boxSize := subContainer.GetWidth() + subContainer.GetHorizontalMargins() + subContainer.GetHorizontalBorderSize()
	return int(float64(container.GetWidth()-container.GetHorizontalPadding()) / float64(boxSize))
}
