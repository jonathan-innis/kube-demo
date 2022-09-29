package models

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/bwagner5/kube-demo/pkg/models/cluster"
	"github.com/bwagner5/kube-demo/pkg/models/grid"
	"github.com/bwagner5/kube-demo/pkg/models/interactive"
	"github.com/bwagner5/kube-demo/pkg/models/node"
	"github.com/bwagner5/kube-demo/pkg/models/pod"
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/style"
)

type Model struct {
	nodeGridModel    grid.Model[node.Model, node.UpdateMsg, node.DeleteMsg]
	podGridModel     grid.Model[pod.Model, pod.UpdateMsg, pod.DeleteMsg]
	clusterModel     cluster.Model
	interactiveModel interactive.Model
	viewType         ViewType
	mode             Mode
	selectedPod      int
	toggleDetails    bool
	stop             chan struct{}
	k8sStateUpdate   chan struct{}
	help             help.Model
	events           <-chan state.Event
	viewport         viewport.Model
}

func NewModel(c *state.Cluster) Model {
	stop := make(chan struct{})
	events := make(chan state.Event, 100)
	c.AddOnChangeObserver(func(evt state.Event) { events <- evt })
	return Model{
		nodeGridModel:    grid.NewModel[node.Model, node.UpdateMsg, node.DeleteMsg](&style.Canvas, node.GridUpdate, node.GridDelete),
		clusterModel:     cluster.NewModel(c),
		interactiveModel: interactive.NewModel(),
		stop:             stop,
		events:           events,
		help:             help.New(),
		viewType:         NodeView,
		mode:             View,
		viewport:         viewport.New(0, 0),
	}
}

type startWatchMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		//m.informerFactory.WaitForCacheSync(m.stopCh)
		return startWatchMsg{}
	}, tea.EnterAltScreen)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			close(m.stop)
			return m, tea.Quit
		case "i":
			switch m.mode {
			case View:
				m.mode = Interactive
			}
		case "esc", "q":
			switch m.mode {
			case Interactive:
				m.mode = View
			default:
				switch m.viewType {
				case NodeView, NodeDetailView:
					m.viewType = NodeView
				case PodView:
					m.viewType = NodeView
				default:
					m.viewType = PodView
				}
			}
		case "enter":
			switch m.viewType {
			case NodeView, PodView:
				m.viewType = PodView
			}
		}
		switch m.viewType {
		case NodeView:
			switch msg.String() {
			case "?":
				m.help.ShowAll = !m.help.ShowAll
			case "v", "d":
				m.viewType = NodeDetailView
			}
		case PodView:
			switch msg.String() {
			case "v", "d":
				m.viewType = PodDetailView
			}
		case NodeDetailView, PodDetailView:
			// Handle keyboard events in the viewport
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	case tea.MouseMsg:
		switch m.viewType {
		case NodeDetailView, PodDetailView:
			// Handle mouse events in the viewport
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	case startWatchMsg, node.UpdateMsg, node.DeleteMsg, cluster.UnboundPodsUpdateMsg:
		cmds = append(cmds, func() tea.Msg {
			select {
			case evt := <-m.events:
				switch evt.Kind {
				case state.NodeKind:
					n := evt.Obj.(*state.Node)
					switch evt.Type {
					case state.Update:
						return node.UpdateMsg{ID: n.Node.Name, Node: n}
					case state.Delete:
						return node.DeleteMsg{ID: n.Node.Name, Node: n}
					}
				case state.PodKind:
					return cluster.UnboundPodsUpdateMsg{}
				}
			case <-m.stop:
				return nil
			}
			return nil
		})
	}

	m.nodeGridModel, cmd = m.nodeGridModel.Update(msg)
	cmds = append(cmds, cmd)
	m.clusterModel, cmd = m.clusterModel.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	var canvas strings.Builder
	physicalWidth, physicalHeight, _ := term.GetSize(int(os.Stdout.Fd()))
	style.Canvas = style.Canvas.MaxWidth(physicalWidth).Width(physicalWidth)
	switch m.viewType {
	case NodeDetailView:
		m.viewport.Height = physicalHeight
		m.viewport.Width = physicalWidth
		m.viewport.SetContent(m.nodeGridModel.Viewport())
		return m.viewport.View()
	case PodDetailView:
		canvas.WriteString("Hello you are in pod detail view now :)")
		return style.Canvas.Render(canvas.String())
	case PodView:
		canvas.WriteString("Hello you are in pod view now :)")
		return style.Canvas.Render(canvas.String())
	case NodeView:
		canvas.WriteString(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.nodeGridModel.View(),
				m.clusterModel.View(),
			),
		)
		_ = physicalHeight - strings.Count(canvas.String(), "\n")
		if m.mode == Interactive {
			return style.Canvas.Render(canvas.String()+strings.Repeat("\n", 0)) + "\n" + m.interactiveModel.View() + "\n" + m.help.View(keyMappings)
		}
		return style.Canvas.Render(canvas.String()+strings.Repeat("\n", 0)) + "\n" + m.help.View(keyMappings)
	}
	return ""
}
