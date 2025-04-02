package source_picker

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func PickSource(client *ingestion.APIClient) (string, error) {
	resp, err := client.ListSources(client.NewApiListSourcesRequest().WithType([]ingestion.SourceType{ingestion.SOURCE_TYPE_JSON, ingestion.SOURCE_TYPE_CSV, ingestion.SOURCE_TYPE_DOCKER, ingestion.SOURCE_TYPE_BIGQUERY}).WithItemsPerPage(100))
	if err != nil {
		return "", err
	}

	items := make([]list.Item, 0, len(resp.Sources))

	for _, source := range resp.Sources {
		items = append(items, item{title: fmt.Sprintf("%s: %s", source.GetType(), source.GetName()), uuid: source.GetSourceID()})
	}

	list := list.New(items, itemDelegate{}, 20, 14)
	list.Title = "Your sources"
	list.SetShowStatusBar(false)
	list.SetFilteringEnabled(false)
	list.Styles.Title = titleStyle
	list.Styles.PaginationStyle = paginationStyle
	list.Styles.HelpStyle = helpStyle

	m := model{list: list}

	if _, err := tea.NewProgram(&m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return m.choice, nil
}

/////////////////// bubbles stuff for list selection

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type item struct {
	title, uuid string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.uuid }
func (i item) FilterValue() string { return i.title }

type model struct {
	list   list.Model
	choice string
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i.uuid)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	return "\n" + m.list.View()
}

var (
	docStyle          = lipgloss.NewStyle().Margin(1, 2)
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)
