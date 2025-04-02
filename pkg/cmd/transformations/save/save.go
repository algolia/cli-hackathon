package save

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
)

type SaveOptions struct {
	Config config.IConfig
	IO     *iostreams.IOStreams

	IngestionClient func() (*ingestion.APIClient, error)

	TransformationPath string
	TransformationName string

	PrintFlags *cmdutil.PrintFlags
}

// NewSaveCmd creates a new transformation setup
func NewSaveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &SaveOptions{
		IO:              f.IOStreams,
		Config:          f.Config,
		IngestionClient: f.IngestionClient,
		PrintFlags:      cmdutil.NewPrintFlags(),
	}

	cmd := &cobra.Command{
		Use:     "save transformation-name",
		Aliases: []string{"n"},
		Short:   "Save transformation",
		Example: heredoc.Doc(`
			# Save transformation 
			$ algolia transfo save <transformation-name>
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TransformationPath = args[0]
			}

			return runSaveCmd(opts)
		},
		Annotations: map[string]string{
			"runInWebCLI": "true",
			"acls":        "addObject,deleteObject,listIndexes,deleteIndex,settings,editSettings",
		},
	}

	opts.PrintFlags.AddFlags(cmd)

	return cmd
}

func runSaveCmd(opts *SaveOptions) error {
	client, err := opts.IngestionClient()
	if err != nil {
		return err
	}

	if opts.TransformationPath == "" {
		opts.TransformationPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("unable to retrieve current working directory: %w", err)
		}
	}

	if _, err := os.Stat(fmt.Sprintf("%s%cpackage.json", opts.TransformationPath, os.PathSeparator)); os.IsNotExist(err) {
		return fmt.Errorf("no transformation found at path '%s'", opts.TransformationPath)
	}

	code, err := os.ReadFile(fmt.Sprintf("%s%cindex.js", opts.TransformationPath, os.PathSeparator))
	if err != nil {
		return fmt.Errorf("unable to find 'index.js' file at path '%s': %w", opts.TransformationPath, err)
	}

	pkg, err := os.ReadFile(fmt.Sprintf("%s%cpackage.json", opts.TransformationPath, os.PathSeparator))
	if err != nil {
		return fmt.Errorf("unable to find 'package.json' file at path '%s': %w", opts.TransformationPath, err)
	}

	var packageJson struct{ Name string }

	if err := json.Unmarshal(pkg, &packageJson); err != nil {
		return fmt.Errorf("unable to read 'package.json' at path '%s': %w", opts.TransformationPath, err)
	}

	opts.IO.StartProgressIndicatorWithLabel(fmt.Sprintf("Saving transformation at path '%s'", opts.TransformationPath))

	_, err = client.CreateTransformation(
		client.NewApiCreateTransformationRequest(
			ingestion.NewEmptyTransformationCreate().
				SetCode(string(code)).
				SetName(packageJson.Name).
				SetDescription("Transformation created from the Algolia CLI tool"),
		))
	if err != nil {
		return err
	}

	opts.IO.StopProgressIndicator()

	return nil
}
