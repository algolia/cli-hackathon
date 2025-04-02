package bubbleinput

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func Prompt(message string) (string, error) {
	ti := textinput.New()
	ti.Placeholder = "..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	m := model{
		message:   message,
		textInput: ti,
		err:       nil,
	}

	if _, err := tea.NewProgram(&m).Run(); err != nil {
		return "", err
	}

	return m.textInput.Value(), nil
}

type (
	errMsg error
)

type model struct {
	message   string
	textInput textinput.Model
	err       error
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
}

func (m *model) View() string {
	return fmt.Sprintf(
		"%s\n\n\n%s",
		m.message,
		m.textInput.View(),
	) + "\n"
}
