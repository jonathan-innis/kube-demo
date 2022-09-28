package model

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
	v1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/model/style"
	"github.com/bwagner5/kube-demo/pkg/model/views"
	"github.com/bwagner5/kube-demo/pkg/state"
)

type k8sStateChange struct{}

type Model struct {
	cluster        *state.Cluster
	storedNodes    []*state.Node
	unboundPods    []*v1.Pod
	viewType       views.ViewType
	selectedNode   int
	selectedPod    int
	podSelection   bool
	toggleDetails  bool
	stop           chan struct{}
	k8sStateUpdate chan struct{}
	help           help.Model
	events         <-chan struct{}
	viewport       viewport.Model
}

func NewModel(cluster *state.Cluster) *Model {
	stop := make(chan struct{})
	events := make(chan struct{}, 100)
	model := &Model{
		cluster:  cluster,
		stop:     stop,
		events:   events,
		help:     help.New(),
		viewType: views.NodeView,
		viewport: viewport.New(0, 0),
	}
	cluster.AddOnChangeObserver(func() { events <- struct{}{} })
	return model
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		//m.informerFactory.WaitForCacheSync(m.stopCh)
		return k8sStateChange{}
	}, tea.EnterAltScreen)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			close(m.stop)
			return m, tea.Quit
		case "esc", "q":
			switch m.viewType {
			case views.NodeView, views.NodeDetailView:
				m.viewType = views.NodeView
			case views.PodView:
				m.viewType = views.NodeView
			default:
				m.viewType = views.PodView
			}
		case "enter":
			switch m.viewType {
			case views.NodeView, views.PodView:
				m.viewType = views.PodView
			}
		}
		switch m.viewType {
		case views.NodeView:
			switch msg.String() {
			case "left", "right", "up", "down":
				m.selectedNode = m.moveCursor(msg)
			case "?":
				m.help.ShowAll = !m.help.ShowAll
			case "v", "d":
				m.viewType = views.NodeDetailView
			}
		case views.PodView:
			switch msg.String() {
			case "v", "d":
				m.viewType = views.PodDetailView
			}
		case views.NodeDetailView, views.PodDetailView:
			// Handle keyboard and mouse events in the viewport
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	case k8sStateChange:
		return m, func() tea.Msg {
			select {
			case <-m.events:
				m.storedNodes = []*state.Node{}
				m.unboundPods = []*v1.Pod{}
				m.cluster.ForEachNode(func(n *state.Node) bool {
					m.storedNodes = append(m.storedNodes, n)
					return true
				})
				m.cluster.ForEachUnboundPod(func(p *v1.Pod) bool {
					m.unboundPods = append(m.unboundPods, p)
					return true
				})
				return k8sStateChange{}
			case <-m.stop:
				return nil
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) moveCursor(key tea.KeyMsg) int {
	totalObjects := len(m.storedNodes)
	perRow := views.GetBoxesPerRow(style.Canvas, style.Node)
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

func (m *Model) View() string {
	var canvas strings.Builder
	physicalWidth, physicalHeight, _ := term.GetSize(int(os.Stdout.Fd()))
	style.Canvas = style.Canvas.MaxWidth(physicalWidth).Width(physicalWidth)
	switch m.viewType {
	case views.NodeDetailView:
		m.viewport.Height = physicalHeight
		m.viewport.Width = physicalWidth

		m.viewport.SetContent(views.GetNodeViewportContent(m.storedNodes[m.selectedNode].Node))
		return m.viewport.View()
	case views.PodDetailView:
		canvas.WriteString("Hello you are in pod detail view now :)")
		return style.Canvas.Render(canvas.String())
	case views.PodView:
		canvas.WriteString("Hello you are in pod view now :)")
		return style.Canvas.Render(canvas.String())
	case views.NodeView:
		canvas.WriteString(
			lipgloss.JoinVertical(
				lipgloss.Left,
				views.Nodes(m.selectedNode, m.storedNodes),
				views.Cluster(m.storedNodes, m.unboundPods),
			),
		)
		_ = physicalHeight - strings.Count(canvas.String(), "\n")
		return style.Canvas.Render(canvas.String()+strings.Repeat("\n", 0)) + "\n" + m.help.View(keyMappings)
	}
	return ""
}
