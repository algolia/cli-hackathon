package bubbleinput

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func Prompt(message string) (string, error) {
	m := initialModel(message)

	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	return m.textInput.CurrentSuggestion(), nil
}

type (
	errMsg error
)

type model struct {
	message   string
	textInput textinput.Model
	err       error
}

func initialModel(message string) model {
	ti := textinput.New()
	ti.Placeholder = "Doing fancy stuff to search FAST"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		message:   message,
		textInput: ti,
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
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

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n\n%s",
		m.message,
		m.textInput.View(),
	) + "\n"
}
