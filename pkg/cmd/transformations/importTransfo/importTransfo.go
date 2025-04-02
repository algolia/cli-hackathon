package importTransfo

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	bubblelist "github.com/algolia/cli/pkg/cmd/transformations/bubble/list"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
			$ algolia transformations import
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

		opts.TransformationID, err = PickTransformation(client)
		if err != nil {
			return err
		}

		opts.IO.StopProgressIndicator()
	}

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

func PickTransformation(client *ingestion.APIClient) (string, error) {
	resp, err := client.ListTransformations(client.NewApiListTransformationsRequest())
	if err != nil {
		return "", err
	}

	items := make([]list.Item, 0, len(resp.Transformations))

	for _, transformation := range resp.Transformations {
		items = append(items, bubblelist.Item{Name: transformation.GetName(), UUID: transformation.GetTransformationID()})
	}

	list := bubblelist.NewBubbleList("transformations", items)

	if _, err := tea.NewProgram(&list).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return list.Choice, nil
}
