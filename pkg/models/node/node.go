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
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/style"
	nodeutils "github.com/bwagner5/kube-demo/pkg/utils/node"
	podutils "github.com/bwagner5/kube-demo/pkg/utils/pod"
)

type Model struct {
	id          string
	Node        *state.Node
	StopWatch   stopwatch.Model
	Seen        bool
	JustCreated bool
	BeenReady   bool
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
		id:          n.Node.Name,
		Node:        n,
		StopWatch:   s,
		Seen:        true,
		JustCreated: true,
		BeenReady:   false,
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
	}
	var cmd tea.Cmd
	m.StopWatch, cmd = m.StopWatch.Update(msg)
	return m, tea.Batch(append(cmds, cmd)...)
}

func (m Model) View(overrides ...grid.ViewOverride) string {
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
	node := style.Node.Copy().BorderBackground(color)
	for _, override := range overrides {
		node = override(node)
	}

	podGrid, daemonSetPodGrid := m.getPodGrids()
	return node.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.Node.Node.Name,
			podGrid.View(),
			style.Separator,
			daemonSetPodGrid.View(),
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

func (m Model) GetViewportContent() string {
	return components.GetNodeViewportContent(m.Node.Node)
}

func (m Model) GetCreationTimestamp() int64 {
	return m.Node.Node.CreationTimestamp.Unix()
}

func (m Model) GetUID() string {
	return string(m.Node.Node.UID)
}

func (m Model) GetStyle() *lipgloss.Style {
	return &style.Node
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

func (m Model) getPodGrids() (grid.Model[pod.Model, pod.UpdateMsg, pod.DeleteMsg], grid.Model[pod.Model, pod.UpdateMsg, pod.DeleteMsg]) {
	podGrid := grid.NewModel[pod.Model, pod.UpdateMsg, pod.DeleteMsg](&style.Node, nil, nil)
	daemonSetPodGrid := grid.NewModel[pod.Model, pod.UpdateMsg, pod.DeleteMsg](&style.Node, nil, nil)

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
