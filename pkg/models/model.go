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
	"github.com/bwagner5/kube-demo/pkg/models/views"
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/style"
)

type Model struct {
	nodeGridModel    *grid.Model[*node.Model, node.UpdateMsg, node.DeleteMsg]
	clusterModel     cluster.Model
	interactiveModel interactive.Model
	viewType         views.Type
	viewMode         views.Mode
	stop             chan struct{}
	help             help.Model
	events           <-chan state.Event
	viewport         viewport.Model
}

func NewModel(c *state.Cluster) Model {
	stop := make(chan struct{})
	events := make(chan state.Event, 100)
	c.AddOnChangeObserver(func(evt state.Event) { events <- evt })
	model := Model{
		nodeGridModel:    grid.NewModel[*node.Model, node.UpdateMsg, node.DeleteMsg](&style.Canvas, &style.Node, node.GridUpdate, node.GridDelete),
		clusterModel:     cluster.NewModel(c),
		interactiveModel: interactive.NewModel(),
		stop:             stop,
		events:           events,
		help:             help.New(),
		viewType:         views.NodeType,
		viewMode:         views.ViewMode,
		viewport:         viewport.New(0, 0),
	}
	return model
}

type startWatchMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		//m.informerFactory.WaitForCacheSync(m.stopCh)
		return startWatchMsg{}
	}, tea.EnterAltScreen, views.ChangeViewMode(views.ViewMode), views.ChangeViewType(views.NodeType))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			close(m.stop)
			return m, tea.Quit
		case "i":
			switch m.viewMode {
			case views.ViewMode:
				cmds = append(cmds, views.ChangeViewMode(views.InteractiveMode))
			}
		case "y":
			switch m.viewType {
			case views.NodeType:
				cmds = append(cmds, views.ChangeViewType(views.NodeYAMLType))
			case views.PodType:
				cmds = append(cmds, views.ChangeViewType(views.PodYAMLType))
			}
		case "j":
			switch m.viewType {
			case views.NodeType:
				cmds = append(cmds, views.ChangeViewType(views.NodeJSONType))
			case views.PodType:
				cmds = append(cmds, views.ChangeViewType(views.PodJSONType))
			}
		case "esc":
			switch m.viewMode {
			case views.InteractiveMode:
				cmds = append(cmds, views.ChangeViewMode(views.ViewMode))
			default:
				switch m.viewType {
				case views.NodeType, views.NodeYAMLType, views.NodeJSONType:
					cmds = append(cmds, views.ChangeViewType(views.NodeType))
				case views.PodType:
					cmds = append(cmds, views.ChangeViewType(views.NodeType))
				default:
					cmds = append(cmds, views.ChangeViewType(views.PodType))
				}
			}
		case "enter":
			switch m.viewType {
			case views.NodeType:
				cmds = append(cmds, views.ChangeViewType(views.PodType))
			}
		case "?":
			m.help.ShowAll = !m.help.ShowAll
		}
		switch m.viewType {
		case views.NodeYAMLType, views.PodYAMLType, views.NodeJSONType, views.PodJSONType:
			// Handle keyboard events in the viewport
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	case tea.MouseMsg:
		switch m.viewType {
		case views.NodeYAMLType, views.PodYAMLType, views.NodeJSONType, views.PodJSONType:
			// Handle mouse events in the viewport
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	case views.ViewTypeChangeMsg:
		m.viewType = msg.ActiveView

		// Set the active cursor
		switch msg.ActiveView {
		case views.NodeType:
			m.nodeGridModel.CursorActive = true
		default:
			m.nodeGridModel.CursorActive = false
		}

		// Set the viewport in detail view
		switch msg.ActiveView {
		case views.NodeYAMLType:
			m.viewport.SetContent(m.nodeGridModel.ActiveModel().GetYAML())
		case views.PodYAMLType:
			m.viewport.SetContent(m.nodeGridModel.ActiveModel().PodGridModel.ActiveModel().GetYAML())
		case views.NodeJSONType:
			m.viewport.SetContent(m.nodeGridModel.ActiveModel().GetJSON())
		case views.PodJSONType:
			m.viewport.SetContent(m.nodeGridModel.ActiveModel().PodGridModel.ActiveModel().GetJSON())
		}
	case views.ViewModeChangeMsg:
		m.viewMode = msg.ActiveMode
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
	case views.NodeYAMLType, views.PodYAMLType, views.NodeJSONType, views.PodJSONType:
		m.viewport.Height = physicalHeight
		m.viewport.Width = physicalWidth
		return m.viewport.View()
	case views.PodType:
		canvas.WriteString(
			lipgloss.JoinVertical(lipgloss.Left,
				m.nodeGridModel.SelectedView(),
				m.clusterModel.View(),
			),
		)
		return style.Canvas.Render(canvas.String())
	case views.NodeType:
		canvas.WriteString(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.nodeGridModel.View(grid.Detail),
				m.clusterModel.View(),
			),
		)
		_ = physicalHeight - strings.Count(canvas.String(), "\n")
		if m.viewMode == views.InteractiveMode {
			return style.Canvas.Render(canvas.String()+strings.Repeat("\n", 0)) + "\n" + m.interactiveModel.View() + "\n" + m.help.View(keyMappings)
		}
		return style.Canvas.Render(canvas.String()+strings.Repeat("\n", 0)) + "\n" + m.help.View(keyMappings)
	}
	return ""
}
