package node

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	corev1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/components"
	"github.com/bwagner5/kube-demo/pkg/models/grid"
	"github.com/bwagner5/kube-demo/pkg/models/metadata"
	"github.com/bwagner5/kube-demo/pkg/models/pod"
	"github.com/bwagner5/kube-demo/pkg/models/views"
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/style"
	nodeutils "github.com/bwagner5/kube-demo/pkg/utils/node"
	podutils "github.com/bwagner5/kube-demo/pkg/utils/pod"
)

type Model struct {
	id           string
	Node         *state.Node
	PodGridModel grid.Model[pod.Model, pod.UpdateMsg, pod.DeleteMsg]
	StopWatch    stopwatch.Model
	Seen         bool
	JustCreated  bool
	BeenReady    bool
}

type UpdateMsg struct {
	ID   string
	Node *state.Node
}

func (m UpdateMsg) GetID() string {
	return m.ID
}

type DeleteMsg struct {
	ID   string
	Node *state.Node
}

func (m DeleteMsg) GetID() string {
	return m.ID
}

func NewModel(n *state.Node) Model {
	s := stopwatch.New()
	return Model{
		id:           n.Node.Name,
		Node:         n,
		PodGridModel: grid.NewModel[pod.Model, pod.UpdateMsg, pod.DeleteMsg](&style.Node, &style.Pod, pod.GridUpdate, pod.GridDelete),
		StopWatch:    s,
		Seen:         true,
		JustCreated:  true,
		BeenReady:    false,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case UpdateMsg:
		if msg.ID != m.id {
			return m, nil
		}
		if !m.BeenReady && nodeutils.GetReadyStatus(msg.Node.Node) == nodeutils.Ready {
			cmds = append(cmds, m.StopWatch.Stop(), func() tea.Msg { return metadata.NodeCompletedMsg{TimeToReady: m.StopWatch.Elapsed()} })
			m.BeenReady = true
		}
		if m.JustCreated && !m.BeenReady {
			cmds = append(cmds, m.StopWatch.Start())
			m.JustCreated = false
		}
		m.Node = msg.Node

		// Update all the pod models based off of the node
		m.PodGridModel.Models = map[string]pod.Model{}
		for k, v := range m.Node.Pods {
			m.PodGridModel.Models[k.String()] = pod.NewModel(v)
		}
	case views.ViewTypeChangeMsg:
		switch msg.ActiveView {
		case views.PodType:
			m.PodGridModel.CursorActive = true
		default:
			m.PodGridModel.CursorActive = false
		}
	}
	var cmd tea.Cmd
	m.PodGridModel, cmd = m.PodGridModel.Update(msg)
	cmds = append(cmds, cmd)
	m.StopWatch, cmd = m.StopWatch.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View(vt grid.ViewType, overrides ...grid.ViewOverride) string {
	var color lipgloss.Color
	readyConditionStatus := nodeutils.GetCondition(m.Node.Node, corev1.NodeReady).Status
	switch {
	case m.Node.Node.Spec.Unschedulable:
		color = style.Orange
	case readyConditionStatus == "False":
		color = style.Red
	case readyConditionStatus == "True":
		color = style.Grey
	default:
		color = style.Yellow
	}

	switch vt {
	case grid.Single:
		return style.Canvas.Copy().BorderBackground(color).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				m.Node.Node.Name,
				m.PodGridModel.View(grid.Detail),
				"\n",
				fmt.Sprintf("\nStatus: %s", nodeutils.GetReadyStatus(m.Node.Node)),
				getMetadata(m),
			),
		)
	default:
		node := style.Node.Copy().BorderBackground(color)
		for _, override := range overrides {
			node = override(node)
		}
		return node.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				m.Node.Node.Name,
				m.PodGridModel.View(grid.Standard),
				"\n",
				progress.New(progress.WithWidth(style.Node.GetWidth()-style.Node.GetHorizontalPadding()), progress.WithScaledGradient("#FF7CCB", "#FDFF8C")).
					ViewAs(float64(m.Node.PodTotalRequests.Cpu().Value())/float64(m.Node.Allocatable.Cpu().Value())),
				progress.New(progress.WithWidth(style.Node.GetWidth()-style.Node.GetHorizontalPadding()), progress.WithScaledGradient("#FF7CCB", "#FDFF8C")).
					ViewAs(float64(m.Node.PodTotalRequests.Memory().Value())/float64(m.Node.Allocatable.Memory().Value())),
				fmt.Sprintf("\nStatus: %s", nodeutils.GetReadyStatus(m.Node.Node)),
				getMetadata(m),
			),
		)
	}

}

func (m Model) GetViewportContent() string {
	return components.MarshalViewportContent(m.Node.Node)
}

func (m Model) GetCreationTimestamp() int64 {
	return m.Node.Node.CreationTimestamp.Unix()
}

func (m Model) GetUID() string {
	return string(m.Node.Node.UID)
}

func getMetadata(m Model) string {
	return strings.Join([]string{
		fmt.Sprintf("Time to Ready: %v", m.StopWatch.View()),
		fmt.Sprintf("Pods: %d", len(m.Node.Pods)),
		fmt.Sprintf("Instance Type: %s", getValueOrDefault(m.Node.Node.Labels, "beta.kubernetes.io/instance-type", "Unknown")),
		fmt.Sprintf("Capacity Type: %s", getValueOrDefault(m.Node.Node.Labels, "karpenter.sh/capacity-type", "Unknown")),
	}, "\n")
}

func getValueOrDefault[K comparable, V any](m map[K]V, k K, d V) V {
	v, ok := m[k]
	if !ok {
		return d
	}
	return v
}

func (m Model) getPodGrids(containerStyle, subContainerStyle *lipgloss.Style) (grid.Model[pod.Model, pod.UpdateMsg, pod.DeleteMsg], grid.Model[pod.Model, pod.UpdateMsg, pod.DeleteMsg]) {
	podGrid := grid.NewModel[pod.Model, pod.UpdateMsg, pod.DeleteMsg](containerStyle, subContainerStyle, nil, nil)
	daemonSetPodGrid := grid.NewModel[pod.Model, pod.UpdateMsg, pod.DeleteMsg](containerStyle, subContainerStyle, nil, nil)

	// Set the max items to show for each pod type
	podGrid.MaxItemsShown = 50
	daemonSetPodGrid.MaxItemsShown = 10

	for k, v := range m.Node.Pods {
		if !podutils.IsOwnedByDaemonSet(v) {
			podGrid.Models[k.String()] = pod.NewModel(v)
		} else {
			daemonSetPodGrid.Models[k.String()] = pod.NewModel(v)
		}
	}
	return podGrid, daemonSetPodGrid
}
