package metadata

import (
	"fmt"
	"math"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/samber/lo"

	"github.com/bwagner5/kube-demo/pkg/style"
)

type NodeCompletedMsg struct {
	TimeToReady time.Duration
}

type Model struct {
	LongestTimeToReady  time.Duration
	ShortestTimeToReady time.Duration
	AverageTimeToReady  time.Duration

	TotalSeconds int64
	Count        int64
}

func NewModel() Model {
	return Model{
		ShortestTimeToReady: math.MaxInt,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case NodeCompletedMsg:
		// Anything less than this value, and we assume that it is too short to be real
		if msg.TimeToReady.Seconds() > 10 {
			if msg.TimeToReady.Seconds() > m.LongestTimeToReady.Seconds() {
				m.LongestTimeToReady = msg.TimeToReady
			}
			if msg.TimeToReady.Seconds() < m.ShortestTimeToReady.Seconds() {
				m.ShortestTimeToReady = msg.TimeToReady
			}
			m.Count++
			m.TotalSeconds += int64(msg.TimeToReady.Seconds())
			if m.Count != 0 {
				m.AverageTimeToReady = lo.Must(time.ParseDuration(fmt.Sprintf("%ds", m.TotalSeconds/m.Count)))
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	shortest := m.ShortestTimeToReady.String()
	if m.ShortestTimeToReady == math.MaxInt {
		shortest = "NaN"
	}
	return lipgloss.JoinHorizontal(lipgloss.Left,
		fmt.Sprintf("Time To Ready %s ", style.Separator),
		fmt.Sprintf("Average - %v", m.AverageTimeToReady),
		"\t",
		fmt.Sprintf("Longest - %v", m.LongestTimeToReady),
		"\t",
		fmt.Sprintf("Shortest - %v", shortest),
	)
}
