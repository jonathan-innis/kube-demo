package grid

import (
	"fmt"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/bwagner5/kube-demo/pkg/components"
	"github.com/bwagner5/kube-demo/pkg/style"
	"github.com/bwagner5/kube-demo/pkg/utils/atomic"
)

type modelUpdateFunc[T Interface[T, U], U, D MessageInterface] func(*Model[T, U, D], U)
type modelDeleteFunc[T Interface[T, U], U, D MessageInterface] func(*Model[T, U, D], D)

type Model[T Interface[T, U], U, D MessageInterface] struct {
	containerStyle    *lipgloss.Style
	subContainerStyle *lipgloss.Style

	Models *atomic.Map[string, T]

	onUpdate modelUpdateFunc[T, U, D]
	onDelete modelDeleteFunc[T, U, D]

	// View-related options
	selected      int
	CursorActive  bool // Defines whether the cursor should respond to keyboard events
	MaxItemsShown int
}

func NewModel[T Interface[T, U], U, D MessageInterface](containerStyle, subContainerStyle *lipgloss.Style, onUpdate modelUpdateFunc[T, U, D], onDelete modelDeleteFunc[T, U, D]) *Model[T, U, D] {
	return &Model[T, U, D]{
		containerStyle:    containerStyle,
		subContainerStyle: subContainerStyle,
		Models:            atomic.NewMap[string, T](),

		onUpdate: onUpdate,
		onDelete: onDelete,

		MaxItemsShown: 50,
	}
}

func NewModelFromModels[T Interface[T, U], U, D MessageInterface](
	containerStyle, subContainerStyle *lipgloss.Style,
	onUpdate modelUpdateFunc[T, U, D], onDelete modelDeleteFunc[T, U, D],
	models *atomic.Map[string, T]) *Model[T, U, D] {
	return &Model[T, U, D]{
		containerStyle:    containerStyle,
		subContainerStyle: subContainerStyle,
		Models:            models,

		onUpdate: onUpdate,
		onDelete: onDelete,

		MaxItemsShown: 50,
	}
}

func (m *Model[T, U, D]) Init() tea.Cmd { return nil }

func (m *Model[T, U, D]) Update(msg tea.Msg) (*Model[T, U, D], tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "right", "up", "down":
			if m.CursorActive {
				m.selected = m.moveCursor(msg)
			}
		}
	case U:
		m.onUpdate(m, msg)
	case D:
		m.onDelete(m, msg)
	}
	newModels := atomic.NewMap[string, T]()
	m.Models.Range(func(k string, v T) {
		updated, cmd := v.Update(msg)
		newModels.Load(k, updated)
		cmds = append(cmds, cmd)
	})
	m.Models = newModels
	return m, tea.Batch(cmds...)
}

func (m *Model[T, U, D]) View(vt ViewType) string {
	var extraInfo string
	listView := m.listView()

	if len(listView) > m.MaxItemsShown {
		extraInfo = fmt.Sprintf("[and %d others]", len(listView)-m.MaxItemsShown)
		listView = listView[:m.MaxItemsShown]
	}
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
				vt,
				func(s lipgloss.Style) lipgloss.Style { return s.BorderBackground(style.SelectedBorder) }),
			)
		} else {
			boxRows[row] = append(boxRows[row], elem.View(vt))
		}
	}
	rows := lo.Map(boxRows, func(row []string, _ int) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, row...)
	})
	rows = append(rows, extraInfo)
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *Model[T, U, D]) SelectedView() string {
	listView := m.listView()
	for i, elem := range listView {
		if i == m.selected {
			return elem.View(Single)
		}
	}
	return ""
}

func (m *Model[T, U, D]) ActiveModel() T {
	return m.listView()[m.selected]
}

func (m *Model[T, U, D]) listView() []T {
	var listView []T
	m.Models.Range(func(k string, v T) {
		listView = append(listView, v)
	})

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

func (m *Model[T, U, D]) moveCursor(key tea.KeyMsg) int {
	totalObjects := m.Models.Len()
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
