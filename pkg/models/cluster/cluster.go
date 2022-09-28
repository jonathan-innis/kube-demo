package cluster

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/components"
	"github.com/bwagner5/kube-demo/pkg/models/metadata"
	"github.com/bwagner5/kube-demo/pkg/models/node"
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/style"
	"github.com/bwagner5/kube-demo/pkg/utils/resources"
)

type Model struct {
	cluster     *state.Cluster
	nodes       map[string]*state.Node
	unboundPods []*v1.Pod
	metadata    metadata.Model
}

func NewModel(cluster *state.Cluster) Model {
	return Model{
		cluster:  cluster,
		nodes:    map[string]*state.Node{},
		metadata: metadata.NewModel(),
	}
}

type UnboundPodsUpdateMsg struct{}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case node.UpdateMsg:
		m.nodes[msg.ID] = msg.Node
	case node.DeleteMsg:
		delete(m.nodes, msg.ID)
	case UnboundPodsUpdateMsg:
		m.unboundPods = []*v1.Pod{}
		m.cluster.ForEachUnboundPod(func(p *v1.Pod) bool {
			m.unboundPods = append(m.unboundPods, p)
			return true
		})
	}
	var cmd tea.Cmd
	m.metadata, cmd = m.metadata.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"Cluster",
		style.Separator,
		m.metadata.View(),
		fmt.Sprintf("Unbound Pods Count: %d", len(m.unboundPods)),
		m.clusterUtilization(),
	)
}

func (m Model) clusterUtilization() string {
	allocatable := v1.ResourceList{}
	used := v1.ResourceList{}
	for _, node := range m.nodes {
		allocatable = resources.Merge(allocatable, node.Allocatable)
		used = resources.Merge(used, node.PodTotalRequests)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left,
		fmt.Sprintf("Utilization %s ", style.Separator),
		components.UtilizationBar("CPU", used.Cpu().Value(), allocatable.Cpu().Value(), progress.WithWidth(style.Node.GetWidth()-style.Node.GetHorizontalPadding())),
		"\t",
		components.UtilizationBar("Memory", used.Memory().Value(), allocatable.Memory().Value(), progress.WithWidth(style.Node.GetWidth()-style.Node.GetHorizontalPadding())),
	)
}
