package interactive

import tea "github.com/charmbracelet/bubbletea"

type Model struct{}

func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(_ tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return "You are interactive mode!"
}
