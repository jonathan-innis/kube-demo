package model

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"golang.org/x/term"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/bwagner5/kube-demo/pkg/model/style"
	"github.com/bwagner5/kube-demo/pkg/model/views"
	"github.com/bwagner5/kube-demo/pkg/state"
	nodeutils "github.com/bwagner5/kube-demo/pkg/utils/node"
)

type k8sStateChange struct{}

type Model struct {
	cluster             *state.Cluster
	storedNodes         map[string]*views.NodeModel
	nodeStartupMetadata *views.NodeStartupMetadata
	unboundPods         []*v1.Pod
	viewType            views.ViewType
	selectedNode        int
	selectedPod         int
	toggleDetails       bool
	stop                chan struct{}
	k8sStateUpdate      chan struct{}
	help                help.Model
	events              <-chan struct{}
	viewport            viewport.Model
}

func NewModel(cluster *state.Cluster) *Model {
	stop := make(chan struct{})
	events := make(chan struct{}, 100)
	model := &Model{
		cluster:             cluster,
		storedNodes:         map[string]*views.NodeModel{},
		nodeStartupMetadata: &views.NodeStartupMetadata{},
		stop:                stop,
		events:              events,
		help:                help.New(),
		viewType:            views.NodeView,
		viewport:            viewport.New(0, 0),
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
			// Handle keyboard events in the viewport
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	case tea.MouseMsg:
		switch m.viewType {
		case views.NodeDetailView, views.PodDetailView:
			// Handle mouse events in the viewport
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	case k8sStateChange:
		cmds = append(cmds, func() tea.Msg {
			select {
			case <-m.events:
				m.unboundPods = []*v1.Pod{}
				nodeNames := sets.NewString(lo.Keys(m.storedNodes)...)
				m.cluster.ForEachNode(func(n *state.Node) bool {
					if _, ok := m.storedNodes[n.Node.Name]; !ok {
						m.storedNodes[n.Node.Name] = views.NewNodeModel(n)
					}
					m.storedNodes[n.Node.Name].Node = n
					nodeNames.Delete(n.Node.Name)
					return true
				})
				// Cleanup old nodes that have been removed from the cluster state
				for name := range nodeNames {
					delete(m.storedNodes, name)
				}
				m.cluster.ForEachUnboundPod(func(p *v1.Pod) bool {
					m.unboundPods = append(m.unboundPods, p)
					return true
				})
				return k8sStateChange{}
			case <-m.stop:
				return nil
			}
		})
	}
	// Update all the node stopwatches
	// Keep track of which stopwatches should be started and stopped and update node uptime metadata
	for _, elem := range m.storedNodes {
		var cmd tea.Cmd
		if !elem.BeenReady && nodeutils.GetReadyStatus(elem.Node.Node) == nodeutils.Ready {
			cmds = append(cmds, elem.StopWatch.Stop())
			elem.BeenReady = true

			if elem.StopWatch.Elapsed().Seconds() > m.nodeStartupMetadata.LongestTimeToReady.Seconds() {
				m.nodeStartupMetadata.LongestTimeToReady = elem.StopWatch.Elapsed()
			}
			if elem.StopWatch.Elapsed().Seconds() < m.nodeStartupMetadata.ShortestTimeToReady.Seconds() {
				m.nodeStartupMetadata.ShortestTimeToReady = elem.StopWatch.Elapsed()
			}
			m.nodeStartupMetadata.Count++
			m.nodeStartupMetadata.TotalSeconds += int64(elem.StopWatch.Elapsed().Seconds())
			if m.nodeStartupMetadata.Count != 0 {
				m.nodeStartupMetadata.AverageTimeToReady = lo.Must(time.ParseDuration(fmt.Sprintf("%ds", m.nodeStartupMetadata.TotalSeconds/m.nodeStartupMetadata.Count)))
			}
		}
		if elem.JustCreated && !elem.BeenReady {
			cmds = append(cmds, elem.StopWatch.Start())
			elem.JustCreated = false
		}
		elem.StopWatch, cmd = elem.StopWatch.Update(msg)
		cmds = append(cmds, cmd)
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
	listView := lo.Values(m.storedNodes)
	sort.SliceStable(listView, func(i, j int) bool {
		iCreated := listView[i].Node.Node.CreationTimestamp.Unix()
		jCreated := listView[j].Node.Node.CreationTimestamp.Unix()
		if iCreated == jCreated {
			return string(listView[i].Node.Node.UID) < string(listView[j].Node.Node.UID)
		}
		return iCreated < jCreated
	})
	var canvas strings.Builder
	physicalWidth, physicalHeight, _ := term.GetSize(int(os.Stdout.Fd()))
	style.Canvas = style.Canvas.MaxWidth(physicalWidth).Width(physicalWidth)
	switch m.viewType {
	case views.NodeDetailView:
		m.viewport.Height = physicalHeight
		m.viewport.Width = physicalWidth

		m.viewport.SetContent(views.GetNodeViewportContent(listView[m.selectedNode].Node.Node))
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
				views.Nodes(m.selectedNode, listView),
				views.Cluster(listView, m.unboundPods, m.nodeStartupMetadata),
			),
		)
		_ = physicalHeight - strings.Count(canvas.String(), "\n")
		return style.Canvas.Render(canvas.String()+strings.Repeat("\n", 0)) + "\n" + m.help.View(keyMappings)
	}
	return ""
}
