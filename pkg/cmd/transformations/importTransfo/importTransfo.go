package importTransfo

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
	"github.com/algolia/cli/pkg/validators"
)

type ImportOptions struct {
	Config config.IConfig
	IO     *iostreams.IOStreams

	IngestionClient func() (*ingestion.APIClient, error)

	PrintFlags *cmdutil.PrintFlags

	TransformationID string
}

// NewImportCmd creates and returns a list command for indices
func NewImportCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &ImportOptions{
		IO:              f.IOStreams,
		Config:          f.Config,
		IngestionClient: f.IngestionClient,
		PrintFlags:      cmdutil.NewPrintFlags(),
	}
	cmd := &cobra.Command{
		Use:     "import <transformationID>",
		Aliases: []string{"i"},
		Args:    validators.NoArgs(),
		Short:   "Import existing transformation in your IDE",
		Example: heredoc.Doc(`
			# Import transformation by ID
			$ algolia transformations import 903a251f-1524-4823-8b7e-81a9376fff0e
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImportCmd(opts, args)
		},
		Annotations: map[string]string{
			"runInWebCLI": "true",
			"acls":        "addObject,deleteObject,listIndexes,deleteIndex,settings,editSettings",
		},
	}

	opts.PrintFlags.AddFlags(cmd)

	return cmd
}

func runImportCmd(opts *ImportOptions, args []string) error {
	client, err := opts.IngestionClient()
	if err != nil {
		return err
	}

	if opts.TransformationID == "" {
		opts.IO.StartProgressIndicatorWithLabel("Listing transformations")

		resp, err := client.ListTransformations(client.NewApiListTransformationsRequest())
		if err != nil {
			return err
		}

		items := make([]list.Item, 0, len(resp.Transformations))

		for _, transformation := range resp.Transformations {
			items = append(items, item{name: transformation.GetName(), uuid: transformation.GetTransformationID()})
		}

		list := list.New(items, list.NewDefaultDelegate(), 20, 0)
		list.Title = "Your transformations"

		m := model{list: list}

		opts.IO.StopProgressIndicator()

		if _, err := tea.NewProgram(&m).Run(); err != nil {
			fmt.Println("Error running program:", err)
			os.Exit(1)
		}

		opts.TransformationID = m.choice
	}

	fmt.Println(opts.TransformationID)

	//if opts.PrintFlags.OutputFlagSpecified() && opts.PrintFlags.OutputFormat != nil {
	//	p, err := opts.PrintFlags.ToPrinter()
	//	if err != nil {
	//		return err
	//	}
	//	return p.Print(opts.IO, nil)
	//}
	//
	//if err := opts.IO.StartPager(); err != nil {
	//	fmt.Fprintf(opts.IO.ErrOut, "error starting pager: %v\n", err)
	//}
	//defer opts.IO.StopPager()
	//
	//table := printers.NewTablePrinter(opts.IO)
	//if table.IsTTY() {
	//	table.AddField("NAME", nil, nil)
	//	table.AddField("ENTRIES", nil, nil)
	//	table.AddField("SIZE", nil, nil)
	//	table.AddField("UPDATED AT", nil, nil)
	//	table.AddField("CREATED AT", nil, nil)
	//	table.AddField("LAST BUILD DURATION", nil, nil)
	//	table.AddField("PRIMARY", nil, nil)
	//	table.AddField("REPLICAS", nil, nil)
	//	table.EndRow()
	//}

	opts.IO.StartProgressIndicatorWithLabel("Fetching transformation")
	res, err := client.GetTransformation(client.NewApiGetTransformationRequest(opts.TransformationID))
	opts.IO.StopProgressIndicator()
	if err != nil {
		return err
	}

	cs := opts.IO.ColorScheme()
	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(
			opts.IO.Out,
			"%s Successfully fetched %s \n%s\n",
			cs.SuccessIcon(),
			res.TransformationID,
			res.Code,
		)
	}

	return nil
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type item struct {
	name, uuid string
}

func (i item) Title() string       { return i.name }
func (i item) Description() string { return i.uuid }
func (i item) FilterValue() string { return i.name }

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
