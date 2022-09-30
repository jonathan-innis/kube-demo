package interactive

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	console string
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.Execute()
		case "backspace":
			m.console = m.console[:len(m.console)-1]
		default:
			m.console += msg.String()
		}
	}
	return m, nil
}

func (m *Model) View() string {
	return "> " + m.console
}

func (m *Model) Execute() {
	splits := strings.Split(m.console, " ")
	cmd := exec.Command(splits[0], splits[1:]...)
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	m.console = ""
}
