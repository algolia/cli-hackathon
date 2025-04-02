package sample

import (
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/cli/pkg/cmd/transformations/setup/transformationpackagetemplate"
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmd/transformations/setup/sourcepicker"
	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
)

type NewOptions struct {
	Config config.IConfig
	IO     *iostreams.IOStreams

	IngestionClient func() (*ingestion.APIClient, error)

	SourceID string

	PrintFlags *cmdutil.PrintFlags
}

// NewSampleCmd creates a command to refresh the sample.js file
func NewSampleCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &NewOptions{
		IO:              f.IOStreams,
		Config:          f.Config,
		IngestionClient: f.IngestionClient,
		PrintFlags:      cmdutil.NewPrintFlags(),
	}

	cmd := &cobra.Command{
		Use:     "sample <sourceID>",
		Aliases: []string{"sample"},
		Short:   "Sample a source",
		Example: heredoc.Doc(`
			# Sample a source 
			$ algolia transfo sample
			$ algolia transfo sample sourceID
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SourceID = args[0]
			}

			return runSampleCmd(opts)
		},
		Annotations: map[string]string{
			"runInWebCLI": "true",
			"acls":        "addObject,deleteObject,listIndexes,deleteIndex,settings,editSettings",
		},
	}

	opts.PrintFlags.AddFlags(cmd)

	return cmd
}

func runSampleCmd(opts *NewOptions) error {
	client, err := opts.IngestionClient()
	if err != nil {
		return err
	}

	if opts.SourceID == "" {
		opts.SourceID, err = sourcepicker.PickSource(client)
		if err != nil {
			return err
		}
	}

	opts.IO.StartProgressIndicatorWithLabel(fmt.Sprintf("Sampling source %s", opts.SourceID))

	resp, err := client.ValidateSourceBeforeUpdate(client.NewApiValidateSourceBeforeUpdateRequest(opts.SourceID, ingestion.NewEmptySourceUpdate()))
	if err != nil {
		return err
	}

	opts.IO.StopProgressIndicator()

	if !resp.HasData() || len(resp.GetData()) == 0 {
		return fmt.Errorf("unable to sample source %s: %s", opts.SourceID, resp.GetMessage())
	}

	if err := transformationpackagetemplate.GenerateSample(resp.GetData()[0]); err != nil {
		return err
	}

	opts.IO.StopProgressIndicator()

	return nil
}
