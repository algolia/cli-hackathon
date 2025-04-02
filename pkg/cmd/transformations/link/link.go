package link

import (
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/cli/pkg/cmd/transformations/link/picker"
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
	"github.com/algolia/cli/pkg/validators"
)

type NewOptions struct {
	Config config.IConfig
	IO     *iostreams.IOStreams

	IngestionClient func() (*ingestion.APIClient, error)

	DestinationID string

	PrintFlags *cmdutil.PrintFlags
}

// NewLinkCmd allows linking a transformation to a destination
func NewLinkCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &NewOptions{
		IO:              f.IOStreams,
		Config:          f.Config,
		IngestionClient: f.IngestionClient,
		PrintFlags:      cmdutil.NewPrintFlags(),
	}

	cmd := &cobra.Command{
		Use:     "link [--destination <destination-id>]",
		Aliases: []string{"ln"},
		Args:    validators.NoArgs(),
		Short:   "Link a transformation to a destination",
		Example: heredoc.Doc(`
			# Link a transformation to a destination from a list
			$ algolia transformations link

			# Link a transformation to the DESTINATION_ID destination
			$ algolia transformations link --destination DESTINATION_ID
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewCmd(opts)
		},
		Annotations: map[string]string{
			"runInWebCLI": "true",
			"acls":        "addObject,deleteObject,listIndexes,deleteIndex,settings,editSettings",
		},
	}

	cmd.Flags().StringVarP(
		&opts.DestinationID,
		"destination",
		"d",
		"",
		"The DestinationID (UUID) to link your transformation to, when omitted, your list of destinations will be prompted",
	)

	opts.PrintFlags.AddFlags(cmd)

	return cmd
}

func runNewCmd(opts *NewOptions) error {
	client, err := opts.IngestionClient()
	if err != nil {
		return err
	}

	if opts.DestinationID == "" {
		opts.DestinationID, err = picker.PickDestination(client)
		if err != nil {
			return err
		}

		if opts.DestinationID == "" {
			return nil
		}
	}

	opts.IO.StartProgressIndicatorWithLabel("Linking to destination")

	transformationID := ""

	updateDestination, err := client.UpdateDestination(client.NewApiUpdateDestinationRequest(
		opts.DestinationID,
		ingestion.NewDestinationUpdate().SetTransformationIDs([]string{transformationID}),
	))
	if err != nil {
		return err
	}

	opts.IO.StopProgressIndicator()

	println(fmt.Sprintf("Transformation '%s' linked to destination '%s'", transformationID, updateDestination.GetDestinationID()))

	return nil
}
