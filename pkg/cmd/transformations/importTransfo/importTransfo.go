package importTransfo

import (
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
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
		Args:    validators.ExactArgs(1),
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
	opts.TransformationID = args[0]

	if err != nil {
		return err
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
