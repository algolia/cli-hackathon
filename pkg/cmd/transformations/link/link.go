package link

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	bubblefile "github.com/algolia/cli/pkg/cmd/transformations/bubble/file"
	"github.com/algolia/cli/pkg/cmd/transformations/link/picker"
	"github.com/spf13/cobra"
	"os"
	"path"

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

	_, err = os.ReadFile("./package.json")
	if err != nil {
		return fmt.Errorf("unable to find 'package.json' file. Please save your transformation first with:\n$ algolia cli transformations save")
	}

	transformationID, err := getInfoFromPackageJSON()
	if err != nil {
		return err
	}

	if transformationID == "" {
		return errors.New("please save your transformation first: algolia cli transformations save")
	}

	if opts.DestinationID == "" {
		opts.DestinationID, err = picker.PickDestination(client, transformationID)
		if err != nil {
			return err
		}

		if opts.DestinationID == "" {
			return nil
		}
	}

	opts.IO.StartProgressIndicatorWithLabel("Linking to destination")

	updateDestination, err := client.UpdateDestination(client.NewApiUpdateDestinationRequest(
		opts.DestinationID,
		ingestion.NewDestinationUpdate().SetTransformationIDs([]string{transformationID}),
	))
	if err != nil {
		return err
	}

	opts.IO.StopProgressIndicator()

	_, _ = fmt.Fprintf(
		opts.IO.Out,
		"Transformation '%s' linked to destination '%s'",
		transformationID,
		updateDestination.GetDestinationID(),
	)

	return nil
}

func getInfoFromPackageJSON() (string, error) {
	// This is a placeholder for the actual implementation
	dir := "./"

	if _, err := os.Stat(path.Join(dir, "package.json")); os.IsNotExist(err) {
		transformationPath := bubblefile.NewBubbleFile()

		dir = path.Dir(transformationPath)
	}

	pkg, err := os.ReadFile(path.Join(dir, "package.json"))
	if err != nil {
		return "", fmt.Errorf("unable to find 'package.json' file at path '%s': %w", dir, err)
	}

	var packageJson struct {
		TransformationID string `json:"transformationID"`
	}

	if err = json.Unmarshal(pkg, &packageJson); err != nil {
		return "", fmt.Errorf("unable to read 'package.json' at path '%s': %w", dir, err)
	}

	return packageJson.TransformationID, nil
}
