package pod

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/components"
	"github.com/bwagner5/kube-demo/pkg/models/grid"
	"github.com/bwagner5/kube-demo/pkg/style"
)

type UpdateMsg struct {
	ID  string
	Pod *v1.Pod
}

func (m UpdateMsg) GetID() string {
	return m.ID
}

type DeleteMsg struct {
	ID  string
	Pod *v1.Pod
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

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case UpdateMsg:
		m.pod = msg.Pod
	}
	return m, nil
}

func (m Model) View(vt grid.ViewType, overrides ...grid.ViewOverride) string {
	var color lipgloss.Color
	switch m.pod.Status.Phase {
	case v1.PodFailed:
		color = style.Red
	case v1.PodPending:
		color = style.Yellow
	default:
		color = style.Teal
	}

	switch vt {
	case grid.Standard:
		return style.Pod.Copy().BorderForeground(color).Render("")
	default:
		pod := style.Node.Copy().BorderForeground(color)
		for _, override := range overrides {
			pod = override(pod)
		}
		return pod.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.pod.Name,
				fmt.Sprintf("Namespace: %s", m.pod.Namespace),
				m.getMetadata(),
			),
		)
	}
}

func (m Model) GetViewportContent() string {
	return components.MarshalViewportContent(m.pod)
}

func (m Model) GetCreationTimestamp() int64 {
	return m.pod.CreationTimestamp.Unix()
}

func (m Model) GetUID() string {
	return string(m.pod.UID)
}

func (m Model) getMetadata() string {
	return strings.Join([]string{
		fmt.Sprintf("Containers: %d", len(m.pod.Spec.Containers)),
	}, "\n")
}
