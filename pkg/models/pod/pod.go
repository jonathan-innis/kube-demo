package pod

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/models/grid"
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/style"
)

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

type Model struct {
	pod *v1.Pod
}

func NewModel(pod *v1.Pod) Model {
	return Model{
		pod: pod,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(_ tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View(...grid.ViewOverride) string {
	var color lipgloss.Color
	switch m.pod.Status.Phase {
	case v1.PodFailed:
		color = style.Red
	case v1.PodPending:
		color = style.Yellow
	default:
		color = style.Teal
	}
	return style.Pod.Copy().BorderForeground(color).Render("")
}

// TODO: Implement getting the viewport content for the pod
func (m Model) GetViewportContent() string {
	return ""
}

func (m Model) GetCreationTimestamp() int64 {
	return m.pod.CreationTimestamp.Unix()
}

func (m Model) GetUID() string {
	return string(m.pod.UID)
}

func (m Model) GetStyle() *lipgloss.Style {
	return &style.Pod
}
