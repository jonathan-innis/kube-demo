package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/model/components"
	"github.com/bwagner5/kube-demo/pkg/model/style"
	"github.com/bwagner5/kube-demo/pkg/utils/resources"
)

type NodeStartupMetadata struct {
	LongestTimeToReady  time.Duration
	ShortestTimeToReady time.Duration
	AverageTimeToReady  time.Duration

	TotalSeconds int64
	Count        int64
}

func Cluster(nodes []*NodeModel, unboundPods []*v1.Pod, metadata *NodeStartupMetadata) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"Cluster",
		style.Separator,
		timeToReadyStats(metadata),
		fmt.Sprintf("Unbound Pods Count: %d", len(unboundPods)),
		clusterUtilization(nodes),
	)
}

func timeToReadyStats(metadata *NodeStartupMetadata) string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		fmt.Sprintf("Time To Ready %s ", style.Separator),
		fmt.Sprintf("Average - %v", metadata.AverageTimeToReady),
		"\t",
		fmt.Sprintf("Longest - %v", metadata.LongestTimeToReady),
		"\t",
		fmt.Sprintf("Shortest - %v", metadata.ShortestTimeToReady),
	)
}

func clusterUtilization(nodes []*NodeModel) string {
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
