package views

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"

	"github.com/bwagner5/kube-demo/pkg/state"
	podutils "github.com/bwagner5/kube-demo/pkg/utils"
)

func (m *Model) daemonsetPods(node *state.Node) string {
	pods := lo.Filter(lo.Values(node.Pods), func(p *corev1.Pod, _ int) bool { return podutils.IsOwnedByDaemonSet(p) })
	sort.SliceStable(pods, func(i, j int) bool {
		iCreated := pods[i].CreationTimestamp.Unix()
		jCreated := pods[j].CreationTimestamp.Unix()
		if iCreated == jCreated {
			return string(pods[i].UID) < string(pods[j].UID)
		}
		return iCreated < jCreated
	})
	return m.getPodsView(pods)
}

func (m *Model) pods(node *state.Node) string {
	pods := lo.Filter(lo.Values(node.Pods), func(p *corev1.Pod, _ int) bool { return !podutils.IsOwnedByDaemonSet(p) })
	sort.SliceStable(pods, func(i, j int) bool {
		iCreated := pods[i].CreationTimestamp.Unix()
		jCreated := pods[j].CreationTimestamp.Unix()
		if iCreated == jCreated {
			return string(pods[i].UID) < string(pods[j].UID)
		}
		return iCreated < jCreated
	})
	return m.getPodsView(pods)
}

func (m *Model) getPodsView(pods []*corev1.Pod) string {
	var boxRows [][]string
	perRow := m.GetBoxesPerRow(nodeStyle, podStyle)
	row := -1
	for i, pod := range pods {
		color := podStyle.GetBorderBottomForeground()
		if i%perRow == 0 {
			boxRows = append(boxRows, []string{})
			row++
		}
		if pod.Status.Phase == corev1.PodPending {
			color = red
		}
		boxRows[row] = append(boxRows[row], podStyle.Copy().BorderForeground(color).Render(""))
	}
	rows := lo.Map(boxRows, func(row []string, _ int) string {
		return lipgloss.JoinHorizontal(lipgloss.Bottom, row...)
	})
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
