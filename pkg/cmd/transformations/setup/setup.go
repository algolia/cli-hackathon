package setup

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
	"github.com/algolia/cli/pkg/printers"
	"github.com/algolia/cli/pkg/validators"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type NewOptions struct {
	Config config.IConfig
	IO     *iostreams.IOStreams

	IngestionClient func() (*ingestion.APIClient, error)

	SourceID string

	PrintFlags *cmdutil.PrintFlags
}

// NewSetupCmd creates a new transformation setup
func NewSetupCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &NewOptions{
		IO:              f.IOStreams,
		Config:          f.Config,
		IngestionClient: f.IngestionClient,
		PrintFlags:      cmdutil.NewPrintFlags(),
	}

	cmd := &cobra.Command{
		Use:     "new transformation-name",
		Aliases: []string{"n"},
		Args:    validators.ExactArgsWithMsg(1, "A transformation name is required"),
		Short:   "New transformation",
		Example: heredoc.Doc(`
			# New transformation 
			$ algolia transfo new transformation-name --source <uuid>
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewCmd(opts)
		},
		Annotations: map[string]string{
			"runInWebCLI": "true",
			"acls":        "addObject,deleteObject,listIndexes,deleteIndex,settings,editSettings",
		},
	}

	cmd.Flags().
		StringVarP(&opts.SourceID, "source", "s", "", "The SourceID (UUID) to fetch sample from, when omitted, your list of source will be prompted")

	opts.PrintFlags.AddFlags(cmd)

	return cmd
}

func runNewCmd(opts *NewOptions) error {
	client, err := opts.IngestionClient()
	if err != nil {
		return err
	}

	if opts.SourceID == "" {
		opts.IO.StartProgressIndicatorWithLabel("Listing sources")

		resp, err := client.ListSources(client.NewApiListSourcesRequest().WithType([]ingestion.SourceType{ingestion.SOURCE_TYPE_JSON, ingestion.SOURCE_TYPE_CSV, ingestion.SOURCE_TYPE_DOCKER, ingestion.SOURCE_TYPE_BIGQUERY}).WithItemsPerPage(100))
		if err != nil {
			return err
		}

		items := make([]list.Item, 0, len(resp.Sources))

		for _, source := range resp.Sources {
			items = append(items, item{title: fmt.Sprintf("%s: %s", source.GetType(), source.GetName()), uuid: source.GetSourceID()})
		}

		list := list.New(items, list.NewDefaultDelegate(), 20, 0)
		list.Title = "Your sources"

		m := model{list: list}

		opts.IO.StopProgressIndicator()

		if _, err := tea.NewProgram(&m).Run(); err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

		opts.SourceID = m.choice
	}

	opts.IO.StartProgressIndicatorWithLabel("Sampling source")
	resp, err := client.ValidateSource(client.NewApiListSourcesRequest().WithType([]ingestion.SourceType{ingestion.SOURCE_TYPE_JSON, ingestion.SOURCE_TYPE_CSV, ingestion.SOURCE_TYPE_DOCKER, ingestion.SOURCE_TYPE_BIGQUERY}).WithItemsPerPage(100))
	if err != nil {
		return err
	}

	fmt.Println(opts.SourceID)

	if opts.PrintFlags.OutputFlagSpecified() && opts.PrintFlags.OutputFormat != nil {
		p, err := opts.PrintFlags.ToPrinter()
		if err != nil {
			return err
		}
		return p.Print(opts.IO, nil)
	}

	if err := opts.IO.StartPager(); err != nil {
		fmt.Fprintf(opts.IO.ErrOut, "error starting pager: %v\n", err)
	}
	defer opts.IO.StopPager()

	table := printers.NewTablePrinter(opts.IO)
	if table.IsTTY() {
		table.AddField("NAME", nil, nil)
		table.AddField("ENTRIES", nil, nil)
		table.AddField("SIZE", nil, nil)
		table.AddField("UPDATED AT", nil, nil)
		table.AddField("CREATED AT", nil, nil)
		table.AddField("LAST BUILD DURATION", nil, nil)
		table.AddField("PRIMARY", nil, nil)
		table.AddField("REPLICAS", nil, nil)
		table.EndRow()
	}

	// for _, index := range res.Items {
	// 	var primary string
	// 	if index.Primary == nil {
	// 		primary = ""
	// 	} else {
	// 		primary = *index.Primary
	// 	}
	// 	updatedAt, err := parseTime(index.UpdatedAt)
	// 	if err != nil {
	// 		return fmt.Errorf("can't parse %s into a time struct", index.UpdatedAt)
	// 	}
	// 	createdAt, err := parseTime(index.CreatedAt)
	// 	if err != nil {
	// 		return fmt.Errorf("can't parse %s into a time struct", index.CreatedAt)
	// 	}
	// 	// Prevent integer overflow
	// 	if index.DataSize < 0 {
	// 		index.DataSize = 0
	// 	}
	// 	table.AddField(index.Name, nil, nil)
	// 	table.AddField(humanize.Comma(int64(index.Entries)), nil, nil)
	// 	table.AddField(humanize.Bytes(uint64(index.DataSize)), nil, nil)
	// 	table.AddField(updatedAt, nil, nil)
	// 	table.AddField(createdAt, nil, nil)
	// 	table.AddField(strconv.Itoa(int(index.LastBuildTimeS))+"s", nil, nil)
	// 	table.AddField(primary, nil, nil)
	// 	table.AddField(fmt.Sprintf("%v", index.Replicas), nil, nil)
	// 	table.EndRow()
	// }
	return table.Render()
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

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
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	return docStyle.Render(m.list.View())
}
