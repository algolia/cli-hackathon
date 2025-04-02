package setup

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/spf13/cobra"

	bubbleinput "github.com/algolia/cli/pkg/cmd/transformations/bubble/input"
	"github.com/algolia/cli/pkg/cmd/transformations/setup/source_picker"
	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
)

type NewOptions struct {
	Config config.IConfig
	IO     *iostreams.IOStreams

	IngestionClient func() (*ingestion.APIClient, error)

	TransformationName string
	SourceID           string
	Sample             map[string]any

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
		Short:   "New transformation",
		Example: heredoc.Doc(`
			# New transformation 
			$ algolia transfo new transformation-name --source <uuid>
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TransformationName = args[0]
			}

			return runNewCmd(opts)
		},
		Annotations: map[string]string{
			"runInWebCLI": "true",
			"acls":        "addObject,deleteObject,listIndexes,deleteIndex,settings,editSettings",
		},
	}

	cmd.Flags().StringVarP(&opts.SourceID, "source", "s", "", "The SourceID (UUID) to fetch sample from, when omitted, your list of source will be prompted")

	opts.PrintFlags.AddFlags(cmd)

	return cmd
}

func runNewCmd(opts *NewOptions) error {
	client, err := opts.IngestionClient()
	if err != nil {
		return err
	}

	if opts.TransformationName == "" {
		opts.TransformationName, err = bubbleinput.Prompt("What's your transformation name?")
		if err != nil {
			return err
		}
	}

	if opts.SourceID == "" {
		opts.IO.StartProgressIndicatorWithLabel("Listing sources")

		opts.SourceID, err = source_picker.PickSource(client)
		if err != nil {
			return err
		}

		opts.IO.StopProgressIndicator()
	}

	opts.IO.StartProgressIndicatorWithLabel(fmt.Sprintf("Sampling source %s", opts.SourceID))

	resp, err := client.ValidateSourceBeforeUpdate(client.NewApiValidateSourceBeforeUpdateRequest(opts.SourceID, ingestion.NewEmptySourceUpdate()))
	if err != nil {
		return err
	}

	if !resp.HasData() || len(resp.GetData()) == 0 {
		return fmt.Errorf("unable to sample source %s: %s", opts.SourceID, resp.GetMessage())
	}

	opts.Sample = resp.GetData()[0]

	if err := os.Mkdir(opts.TransformationName, 0o750); err != nil {
		return err
	}

	return nil
}
