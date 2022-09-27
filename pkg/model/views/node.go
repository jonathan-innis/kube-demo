package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/model/style"
	"github.com/bwagner5/kube-demo/pkg/state"
	nodeutils "github.com/bwagner5/kube-demo/pkg/utils/node"
)

func Nodes(selectedNode int, nodes []*state.Node) string {
	var boxRows [][]string
	row := -1
	perRow := GetBoxesPerRow(style.Canvas, style.Node)
	for i, node := range nodes {
		var color lipgloss.Color
		readyConditionStatus := nodeutils.GetCondition(node.Node, corev1.NodeReady).Status
		switch {
		case node.Node.Spec.Unschedulable:
			color = style.Orange
		case readyConditionStatus == "False":
			color = style.Red
		case readyConditionStatus == "True":
			color = style.Grey
		default:
			color = style.Yellow
		}
		if i == selectedNode {
			color = style.SelectedNodeBorder
		}
		box := style.Node.Copy().BorderBackground(color).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				node.Node.Name,
				Pods(node),
				"---------",
				DaemonSetPods(node),
				"\n",
				progress.New(progress.WithWidth(style.Node.GetWidth()-style.Node.GetHorizontalPadding()), progress.WithScaledGradient("#FF7CCB", "#FDFF8C")).
					ViewAs(float64(node.PodTotalRequests.Cpu().Value())/float64(node.Allocatable.Cpu().Value())),
				progress.New(progress.WithWidth(style.Node.GetWidth()-style.Node.GetHorizontalPadding()), progress.WithScaledGradient("#FF7CCB", "#FDFF8C")).
					ViewAs(float64(node.PodTotalRequests.Memory().Value())/float64(node.Allocatable.Memory().Value())),
				fmt.Sprintf("\nReady: %s", nodeutils.GetCondition(node.Node, corev1.NodeReady).Status),
				getMetadata(node),
			),
		)
		if i%perRow == 0 {
			row++
			boxRows = append(boxRows, []string{})
		}
		boxRows[row] = append(boxRows[row], box)
	}
	rows := lo.Map(boxRows, func(row []string, _ int) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, row...)
	})
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func getMetadata(node *state.Node) string {
	return strings.Join([]string{
		fmt.Sprintf("Pods: %d", len(node.Pods)),
		fmt.Sprintf("Instance Type: %s", getValueOrDefault(node.Node.Labels, "beta.kubernetes.io/instance-type", "Unknown")),
		fmt.Sprintf("Capacity Type: %s", getValueOrDefault(node.Node.Labels, "karpenter.sh/capacity-type", "Unknown")),
	}, "\n")
}

func getValueOrDefault[K comparable, V any](m map[K]V, k K, d V) V {
	v, ok := m[k]
	if !ok {
		return d
	}
	return v
}
