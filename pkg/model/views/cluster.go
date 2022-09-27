package views

import (
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/model/style"
	"github.com/bwagner5/kube-demo/pkg/state"
	"github.com/bwagner5/kube-demo/pkg/utils/resources"
)

func Cluster(nodes []*state.Node) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"Cluster Utilization\n",
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
		progress.New(progress.WithWidth(style.Node.GetWidth()-style.Node.GetHorizontalPadding()), progress.WithScaledGradient("#FF7CCB", "#FDFF8C")).
			ViewAs(float64(used.Cpu().Value())/float64(allocatable.Cpu().Value())),
		"   ",
		progress.New(progress.WithWidth(style.Node.GetWidth()-style.Node.GetHorizontalPadding()), progress.WithScaledGradient("#FF7CCB", "#FDFF8C")).
			ViewAs(float64(used.Memory().Value())/float64(allocatable.Memory().Value())),
	)
}
