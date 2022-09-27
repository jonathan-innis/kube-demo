package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"
	"golang.org/x/term"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nodeutils "github.com/bwagner5/kube-demo/pkg/node"
	"github.com/bwagner5/kube-demo/pkg/state"
)

var canvasStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

const (
	white  = lipgloss.Color("#FFFFFF")
	black  = lipgloss.Color("#000000")
	orange = lipgloss.Color("#FFA500")
	pink   = lipgloss.Color("#F87575")
	teal   = lipgloss.Color("#27CEBD")
	grey   = lipgloss.Color("#6C7D89")
	yellow = lipgloss.Color("#FFFF00")
	red    = lipgloss.Color("#FF0000")
)

var nodeBorder = grey
var selectedNodeBorder = pink
var defaultPodBorder = teal

var nodeStyle = lipgloss.NewStyle().
	Align(lipgloss.Left).
	Foreground(white).
	Border(lipgloss.HiddenBorder(), true).
	BorderBackground(nodeBorder).
	Margin(1).
	Padding(0, 1, 0, 1).
	Height(10).
	Width(30)

var podStyle = lipgloss.NewStyle().
	Align(lipgloss.Bottom).
	Foreground(white).
	Border(lipgloss.NormalBorder(), true).
	BorderForeground(defaultPodBorder).
	Margin(0).
	Padding(0).
	Height(0).
	Width(1)

type keyMap map[string]key.Binding

var keyMappings = keyMap{
	"Move": key.NewBinding(
		key.WithKeys("up", "down", "left", "right"),
		key.WithHelp("↑/↓/←/→", "move"),
	),
	"Help": key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	"Quit": key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k["Move"], k["Quit"], k["Help"]}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k["Move"], k["Help"], k["Quit"]},
	}
}

type k8sStateChange struct{}

type Model struct {
	cluster        *state.Cluster
	storedNodes    []state.Node
	selectedNode   int
	selectedPod    int
	podSelection   bool
	details        bool
	stop           chan struct{}
	k8sStateUpdate chan struct{}
	help           help.Model
	events         <-chan struct{}
	viewport       viewport.Model
}

func New(config *rest.Config, cluster *state.Cluster) *Model {
	stop := make(chan struct{})
	events := make(chan struct{}, 100)
	model := &Model{
		cluster:  cluster,
		stop:     stop,
		events:   events,
		help:     help.New(),
		viewport: viewport.New(0, 0),
	}
	cluster.AddOnChangeObserver(func() { events <- struct{}{} })
	return model
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		//m.informerFactory.WaitForCacheSync(m.stopCh)
		return k8sStateChange{}
	}, tea.EnterAltScreen)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			close(m.stop)
			return m, tea.Quit
		case "left", "right", "up", "down":
			m.selectedNode = m.moveCursor(msg)
		case "enter":
			m.details = !m.details
		case "?":
			m.help.ShowAll = !m.help.ShowAll
		}
	case k8sStateChange:
		return m, func() tea.Msg {
			select {
			case <-m.events:
				m.storedNodes = []state.Node{}
				m.cluster.ForEachNode(func(n *state.Node) bool {
					m.storedNodes = append(m.storedNodes, *n)
					return true
				})
				return k8sStateChange{}
			case <-m.stop:
				return nil
			}
		}
	}
	return m, nil
}

func (m *Model) moveCursor(key tea.KeyMsg) int {
	totalObjects := len(m.storedNodes)
	perRow := m.GetBoxesPerRow(canvasStyle, nodeStyle)
	switch key.String() {
	case "right":
		rowNum := m.selectedNode / perRow
		index := m.selectedNode + 1
		if index >= totalObjects {
			return index - index%perRow
		}
		return rowNum*perRow + index%perRow
	case "left":
		rowNum := m.selectedNode / perRow
		index := rowNum*perRow + mod(m.selectedNode-1, perRow)
		if index >= totalObjects {
			return totalObjects - 1
		}
		return index
	case "up":
		index := m.selectedNode - perRow
		col := mod(index, perRow)
		bottomRow := totalObjects / perRow
		if index < 0 {
			newPos := bottomRow*perRow + col
			if newPos >= totalObjects {
				return newPos - perRow
			}
			return bottomRow*perRow + col
		}
		return index
	case "down":
		index := m.selectedNode + perRow
		if index >= totalObjects {
			return index % perRow
		}
		return index
	}
	return 0
}

// mod perform the modulus calculation
// in go, the % operator is the remainder rather than the modulus
func mod(a, b int) int {
	return (a%b + b) % b
}

func (m *Model) View() string {
	physicalWidth, physicalHeight, _ := term.GetSize(int(os.Stdout.Fd()))
	if m.details {
		m.viewport.Height = physicalHeight
		m.viewport.Width = physicalWidth

		out, err := yaml.Marshal(m.storedNodes[m.selectedNode].Node.Spec)
		if err == nil {
			m.viewport.SetContent(string(out))
		}
		if err != nil {
			panic(err)
		}
		return m.viewport.View()
	}
	canvasStyle = canvasStyle.MaxWidth(physicalWidth).Width(physicalWidth)
	var canvas strings.Builder
	canvas.WriteString(m.nodes())
	_ = physicalHeight - strings.Count(canvas.String(), "\n")
	return canvasStyle.Render(canvas.String()+strings.Repeat("\n", 0)) + "\n" + m.help.View(keyMappings)
}

func (m *Model) GetBoxesPerRow(container lipgloss.Style, subContainer lipgloss.Style) int {
	boxSize := subContainer.GetWidth() + subContainer.GetHorizontalMargins() + subContainer.GetHorizontalBorderSize()
	return int(float64(container.GetWidth()-container.GetHorizontalPadding()) / float64(boxSize))
}

func (m *Model) nodes() string {
	var boxRows [][]string
	row := -1
	perRow := m.GetBoxesPerRow(canvasStyle, nodeStyle)
	for i, node := range m.storedNodes {
		color := nodeStyle.GetBorderBottomBackground()
		readyConditionStatus := nodeutils.GetCondition(node.Node, corev1.NodeReady).Status
		switch {
		case node.Node.Spec.Unschedulable:
			color = orange
		case readyConditionStatus == "False":
			color = red
		case readyConditionStatus == "True":
			color = grey
		default:
			color = yellow
		}
		if i == m.selectedNode {
			color = selectedNodeBorder
		}
		box := nodeStyle.Copy().BorderBackground(color).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				node.Node.Name,
				m.pods(node, nodeStyle),
				progress.New(progress.WithWidth(nodeStyle.GetWidth()-nodeStyle.GetHorizontalPadding()), progress.WithScaledGradient("#FF7CCB", "#FDFF8C")).
					ViewAs(float64(node.PodTotalRequests.Cpu().Value())/float64(node.Allocatable.Cpu().Value())),
				progress.New(progress.WithWidth(nodeStyle.GetWidth()-nodeStyle.GetHorizontalPadding()), progress.WithScaledGradient("#FF7CCB", "#FDFF8C")).
					ViewAs(float64(node.PodTotalRequests.Memory().Value())/float64(node.Allocatable.Memory().Value())),
				fmt.Sprintf("\nReady: %s", nodeutils.GetCondition(node.Node, corev1.NodeReady).Status),
				fmt.Sprintf("Pods: %d", len(node.Pods)),
				fmt.Sprintf("Instance Type: %s", node.Node.Labels["beta.kubernetes.io/instance-type"]),
				fmt.Sprintf("Capacity Type: %s", node.Node.Labels["karpenter.sh/capacity-type"]),
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

func (m *Model) pods(node state.Node, nodeStyle lipgloss.Style) string {
	var boxRows [][]string
	perRow := m.GetBoxesPerRow(nodeStyle, podStyle)
	pods := lo.MapToSlice(node.Pods, func(_ types.NamespacedName, v *corev1.Pod) *corev1.Pod { return v })
	sort.SliceStable(pods, func(i, j int) bool {
		iCreated := pods[i].CreationTimestamp.Unix()
		jCreated := pods[j].CreationTimestamp.Unix()
		if iCreated == jCreated {
			return string(pods[i].UID) < string(pods[j].UID)
		}
		return iCreated < jCreated
	})
	row := -1
	for i, pod := range pods {
		color := podStyle.GetBorderBottomForeground()
		if i%perRow == 0 {
			boxRows = append(boxRows, []string{})
			row++
		}
		for _, o := range pod.OwnerReferences {
			if o.Kind == "DaemonSet" {
				//color = yellow
			}
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

func main() {
	ctx, _ := context.WithCancel(context.Background())
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		log.Fatalf("could not initialize kubeconfig: %v", err)
	}

	cluster := StartControllers(ctx, config)
	p := tea.NewProgram(New(config, cluster))
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func StartControllers(ctx context.Context, config *rest.Config) *state.Cluster {
	kubeClient, err := client.New(config, client.Options{})
	if err != nil {
		log.Fatalf("could not initialize kube-client: %v", err)
	}

	cluster := state.NewCluster(&clock.RealClock{}, kubeClient)
	manager := state.NewManagerOrDie(ctx, config)
	if err := state.Register(ctx, manager, cluster); err != nil {
		log.Fatalf("%v", err)
	}
	go func() {
		if err := manager.Start(ctx); err != nil {
			log.Fatalf("%v", err)
		}
	}()
	return cluster
}
