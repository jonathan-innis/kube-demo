package node_grid

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/bwagner5/kube-demo/pkg/components"
	"github.com/bwagner5/kube-demo/pkg/models/node"
	"github.com/bwagner5/kube-demo/pkg/style"
)

type Model struct {
	nodeModels   map[string]node.Model
	selectedNode int
}

func NewModel() Model {
	return Model{
		nodeModels: map[string]node.Model{},
	}
}

type SelectedNodeChangedMsg struct {
	selected int
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "right", "up", "down":
			m.selectedNode = m.moveCursor(msg)
		}
	case node.UpdateMsg:
		if _, ok := m.nodeModels[msg.ID]; !ok {
			m.nodeModels[msg.ID] = node.NewModel(msg.Node)
		}
	case node.DeleteMsg:
		delete(m.nodeModels, msg.ID)
	}
	for k, v := range m.nodeModels {
		m.nodeModels[k], cmd = v.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	listView := m.listView()
	var boxRows [][]string
	row := -1
	perRow := components.GetBoxesPerRow(style.Canvas, style.Node)
	for i, n := range listView {
		if i%perRow == 0 {
			row++
			boxRows = append(boxRows, []string{})
		}
		if i == m.selectedNode {
			boxRows[row] = append(boxRows[row], n.View(
				func(s lipgloss.Style) lipgloss.Style { return s.BorderBackground(style.SelectedNodeBorder) }),
			)
		} else {
			boxRows[row] = append(boxRows[row], n.View())
		}
	}
	rows := lo.Map(boxRows, func(row []string, _ int) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, row...)
	})
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) Viewport() string {
	listView := lo.Values(m.nodeModels)
	return components.GetNodeViewportContent(listView[m.selectedNode].Node.Node)
}

func (m *Model) listView() []node.Model {
	listView := lo.Values(m.nodeModels)
	sort.SliceStable(listView, func(i, j int) bool {
		iCreated := listView[i].Node.Node.CreationTimestamp.Unix()
		jCreated := listView[j].Node.Node.CreationTimestamp.Unix()
		if iCreated == jCreated {
			return string(listView[i].Node.Node.UID) < string(listView[j].Node.Node.UID)
		}
		return iCreated < jCreated
	})
	return listView
}

func (m *Model) moveCursor(key tea.KeyMsg) int {
	totalObjects := len(m.nodeModels)
	perRow := components.GetBoxesPerRow(style.Canvas, style.Node)
	switch key.String() {
	case "right":
		rowNum := m.selectedNode / perRow
		index := m.selectedNode + 1
		if index >= totalObjects {
			return index - index%perRow
		}
		return rowNum*perRow + index%perRow
	case "left":
		rowNum := m.selectedNode / perRow
		index := rowNum*perRow + mod(m.selectedNode-1, perRow)
		if index >= totalObjects {
			return totalObjects - 1
		}
		return index
	case "up":
		index := m.selectedNode - perRow
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
		index := m.selectedNode + perRow
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
