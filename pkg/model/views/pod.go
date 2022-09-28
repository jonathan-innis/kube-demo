package views

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/model/style"
	"github.com/bwagner5/kube-demo/pkg/state"
	podutils "github.com/bwagner5/kube-demo/pkg/utils/pod"
)

func DaemonSetPods(node *state.Node) string {
	pods := lo.Filter(lo.Values(node.Pods), func(p *corev1.Pod, _ int) bool { return podutils.IsOwnedByDaemonSet(p) })
	sort.SliceStable(pods, func(i, j int) bool {
		iCreated := pods[i].CreationTimestamp.Unix()
		jCreated := pods[j].CreationTimestamp.Unix()
		if iCreated == jCreated {
			return string(pods[i].UID) < string(pods[j].UID)
		}
		return iCreated < jCreated
	})
	return getPodsView(pods)
}

func Pods(node *state.Node) string {
	pods := lo.Filter(lo.Values(node.Pods), func(p *corev1.Pod, _ int) bool { return !podutils.IsOwnedByDaemonSet(p) })
	sort.SliceStable(pods, func(i, j int) bool {
		iCreated := pods[i].CreationTimestamp.Unix()
		jCreated := pods[j].CreationTimestamp.Unix()
		if iCreated == jCreated {
			return string(pods[i].UID) < string(pods[j].UID)
		}
		return iCreated < jCreated
	})
	if len(pods) > 50 {
		return fmt.Sprintf("%s\n[and %d more pods]", getPodsView(pods[:50]), len(pods)-50)
	}
	return getPodsView(pods)
}

func getPodsView(pods []*corev1.Pod) string {
	var boxRows [][]string
	perRow := GetBoxesPerRow(style.Node, style.Pod)
	row := -1
	for i, pod := range pods {
		if i%perRow == 0 {
			boxRows = append(boxRows, []string{})
			row++
		}
		var color lipgloss.Color
		switch pod.Status.Phase {
		case corev1.PodFailed:
			color = style.Red
		case corev1.PodPending:
			color = style.Yellow
		default:
			color = style.Teal
		}
		boxRows[row] = append(boxRows[row], style.Pod.Copy().BorderForeground(color).Render(""))
	}
	rows := lo.Map(boxRows, func(row []string, _ int) string {
		return lipgloss.JoinHorizontal(lipgloss.Bottom, row...)
	})
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
