package grid

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/bwagner5/kube-demo/pkg/components"
	"github.com/bwagner5/kube-demo/pkg/style"
)

type modelUpdateFunc[T Interface[T, U], U, D MessageInterface] func(*Model[T, U, D], U)
type modelDeleteFunc[T Interface[T, U], U, D MessageInterface] func(*Model[T, U, D], D)

type Model[T Interface[T, U], U, D MessageInterface] struct {
	containerStyle    *lipgloss.Style
	subContainerStyle *lipgloss.Style

	Models map[string]T

	onUpdate modelUpdateFunc[T, U, D]
	onDelete modelDeleteFunc[T, U, D]
	selected int
}

func NewModel[T Interface[T, U], U, D MessageInterface](containerStyle *lipgloss.Style, onUpdate modelUpdateFunc[T, U, D], onDelete modelDeleteFunc[T, U, D]) Model[T, U, D] {
	subContainer := *new(T)
	return Model[T, U, D]{
		containerStyle:    containerStyle,
		subContainerStyle: subContainer.GetStyle(),
		Models:            map[string]T{},

		onUpdate: onUpdate,
		onDelete: onDelete,
	}
}

func (m Model[T, U, D]) Init() tea.Cmd { return nil }

func (m Model[T, U, D]) Update(msg tea.Msg) (Model[T, U, D], tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "right", "up", "down":
			m.selected = m.moveCursor(msg)
		}
	case U:
		m.onUpdate(&m, msg)
	case D:
		m.onDelete(&m, msg)
	}
	for k, v := range m.Models {
		m.Models[k], cmd = v.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model[T, U, D]) View() string {
	listView := m.listView()
	var boxRows [][]string
	row := -1
	perRow := components.GetBoxesPerRow(*m.containerStyle, *m.subContainerStyle)
	for i, elem := range listView {
		if i%perRow == 0 {
			row++
			boxRows = append(boxRows, []string{})
		}
		if i == m.selected {
			boxRows[row] = append(boxRows[row], elem.View(
				func(s lipgloss.Style) lipgloss.Style { return s.BorderBackground(style.SelectedBorder) }),
			)
		} else {
			boxRows[row] = append(boxRows[row], elem.View())
		}
	}
	rows := lo.Map(boxRows, func(row []string, _ int) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, row...)
	})
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model[T, U, D]) Viewport() string {
	listView := lo.Values(m.Models)
	return listView[m.selected].GetViewportContent()
}

func (m Model[T, U, D]) listView() []T {
	listView := lo.Values(m.Models)

	sort.SliceStable(listView, func(i, j int) bool {
		iCreated := listView[i].GetCreationTimestamp()
		jCreated := listView[j].GetCreationTimestamp()
		if iCreated == jCreated {
			return listView[i].GetUID() < listView[j].GetUID()
		}
		return iCreated < jCreated
	})
	return listView
}

func (m Model[T, U, D]) moveCursor(key tea.KeyMsg) int {
	totalObjects := len(m.Models)
	perRow := components.GetBoxesPerRow(*m.containerStyle, *m.subContainerStyle)
	switch key.String() {
	case "right":
		rowNum := m.selected / perRow
		index := m.selected + 1
		if index >= totalObjects {
			return index - index%perRow
		}
		return rowNum*perRow + index%perRow
	case "left":
		rowNum := m.selected / perRow
		index := rowNum*perRow + mod(m.selected-1, perRow)
		if index >= totalObjects {
			return totalObjects - 1
		}
		return index
	case "up":
		index := m.selected - perRow
		col := mod(index, perRow)
		bottomRow := totalObjects / perRow
		if index < 0 {
			newPos := bottomRow*perRow + col
			if newPos >= totalObjects {
				return newPos - perRow
			}
			return bottomRow*perRow + col
		}
		return index
	case "down":
		index := m.selected + perRow
		if index >= totalObjects {
			return index % perRow
		}
		return index
	}
	return 0
}

// mod perform the modulus calculation
// in go, the % operator is the remainder rather than the modulus
func mod(a, b int) int {
	return (a%b + b) % b
}
