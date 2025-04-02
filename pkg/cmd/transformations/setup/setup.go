package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

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

	OutputDirectory    string
	TransformationName string
	SourceID           string
	SampleFile         string
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
			$ algolia transfo new <transformation-name>
			$ algolia transfo new <transformation-name> --source <uuid>
			$ algolia transfo new <transformation-name> --source <uuid> --file <path-to-file.json>
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

	cmd.Flags().StringVarP(&opts.SourceID, "source", "s", "", "The SourceID (UUID) to fetch sample from, when omitted, your list of source will be prompted.")
	cmd.Flags().StringVarP(&opts.SampleFile, "file", "f", "", "Path to the file containing a sample JSON object to run your transformation against.")

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

	if opts.SampleFile == "" {
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
	} else {
		data, err := os.ReadFile(opts.SampleFile)
		if err != nil {
			return fmt.Errorf("unable to open file %s: %w", opts.SampleFile, err)
		}

		err = json.Unmarshal(data, &opts.Sample)
		if err != nil {
			return err
		}
	}

	opts.OutputDirectory = fmt.Sprintf("output%c%s", os.PathSeparator, opts.TransformationName)

	if _, err := os.Stat(opts.OutputDirectory); !os.IsNotExist(err) {
		opts.OutputDirectory = fmt.Sprintf("%s-%d", opts.OutputDirectory, time.Now().Unix())

		fmt.Fprintf(opts.IO.Out, "\nDirectory or file with name '%s' already exist, transformation will be saved in '%s'....\n", opts.TransformationName, opts.OutputDirectory)
	}

	if err := os.MkdirAll(opts.OutputDirectory, 0o750); err != nil {
		return fmt.Errorf("unable to create transformation folder with name '%s': %w", opts.OutputDirectory, err)
	}

	return nil
}
