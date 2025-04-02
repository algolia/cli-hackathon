package picker

import (
	bubblelist "github.com/algolia/cli/pkg/cmd/transformations/bubble/list"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Decision string

const (
	Keep    Decision = "keep"
	Replace Decision = "replace"
	Clone   Decision = "clone"
)

func Decide(text string, decisions []bubblelist.Item) (Decision, error) {
	items := make([]list.Item, 0, len(decisions))

	for _, decision := range decisions {
		items = append(items, decision)
	}

	destinations := bubblelist.NewBubbleList(text, items)

	if _, err := tea.NewProgram(&destinations).Run(); err != nil {
		return "", err
	}

	return Decision(destinations.Choice), nil
}
