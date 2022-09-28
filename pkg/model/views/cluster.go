package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/model/components"
	"github.com/bwagner5/kube-demo/pkg/model/style"
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/utils/resources"
)

func Cluster(nodes []*state.Node, unboundPods []*v1.Pod) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"Cluster",
		style.Separator,
		fmt.Sprintf("Unbound Pods Count: %d", len(unboundPods)),
		clusterUtilization(nodes),
	)
}

func clusterUtilization(nodes []*state.Node) string {
	allocatable := v1.ResourceList{}
	used := v1.ResourceList{}
	for _, node := range nodes {
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
