package model

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
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
	selectedNode   int
	selectedPod    int
	podSelection   bool
	details        bool
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			close(m.stop)
			return m, tea.Quit
		case "left", "right", "up", "down":
			m.selectedNode = m.moveCursor(msg)
		case "enter":
			m.details = !m.details
		case "?":
			m.help.ShowAll = !m.help.ShowAll
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
	return m, nil
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
	physicalWidth, physicalHeight, _ := term.GetSize(int(os.Stdout.Fd()))
	if m.details {
		m.viewport.Height = physicalHeight
		m.viewport.Width = physicalWidth

		out, err := yaml.Marshal(m.storedNodes[m.selectedNode].Node.Spec)
		if err == nil {
			m.viewport.SetContent(string(out))
		}
		if err != nil {
			panic(err)
		}
		return m.viewport.View()
	}
	style.Canvas = style.Canvas.MaxWidth(physicalWidth).Width(physicalWidth)
	var canvas strings.Builder
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
