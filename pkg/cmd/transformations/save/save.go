package save

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/spf13/cobra"

	bubblefile "github.com/algolia/cli/pkg/cmd/transformations/bubble/file"
	"github.com/algolia/cli/pkg/cmd/transformations/setup/transformationpackagetemplate"
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

	dir := "./"

	if opts.TransformationPath == "" {
		// check if there is a index.js already
		if _, err = os.Stat("index.js"); os.IsNotExist(err) {
			opts.TransformationPath = bubblefile.NewBubbleFile()
			dir = path.Dir(opts.TransformationPath)
		} else {
			opts.TransformationPath = "index.js"
		}
	}

	if path.Ext(opts.TransformationPath) != ".js" {
		return fmt.Errorf("please provide a valid javascript file, '%s' given", opts.TransformationPath)
	}

	code, err := os.ReadFile("index.js")
	if err != nil {
		return fmt.Errorf("unable to find 'index.js' file at path '%s': %w", dir, err)
	}

	pkg, err := os.ReadFile(path.Join(dir, "package.json"))
	if err != nil {
		return fmt.Errorf("unable to find 'package.json' file at path '%s': %w", dir, err)
	}

	var packageJson struct {
		Name             string `json:"name"`
		TransformationID string `json:"transformationID"`
	}

	if err = json.Unmarshal(pkg, &packageJson); err != nil {
		return fmt.Errorf("unable to read 'package.json' at path '%s': %w", dir, err)
	}

	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("hackathon:hackathon"))

	// Create a new transfo
	if packageJson.TransformationID == "" {
		opts.IO.StartProgressIndicatorWithLabel(fmt.Sprintf("Saving transformation at path '%s'", opts.TransformationPath))

		resp, err := client.CreateTransformation(
			client.NewApiCreateTransformationRequest(
				ingestion.NewTransformationCreate(string(code), packageJson.Name).
					SetDescription("Transformation created from the Algolia CLI tool"),
			))
		if err != nil {
			return err
		}

		respGlobal, err := client.CreateTransformation(
			client.NewApiCreateTransformationRequest(
				ingestion.NewTransformationCreate(string(code), packageJson.Name).
					SetDescription("Transformation created from the Algolia CLI tool"),
			), ingestion.WithHeaderParam("Authorization", basicAuth))
		if err != nil {
			return err
		}

		err = transformationpackagetemplate.RefreshPackageJson(packageJson.Name, resp.GetTransformationID(), respGlobal.GetTransformationID())
		if err != nil {
			return fmt.Errorf("unable to refresh package.json: %w", err)
		}

		opts.IO.StopProgressIndicator()

		return nil
	}

	// Update an existing transfo
	opts.IO.StartProgressIndicatorWithLabel(fmt.Sprintf("Updating transformation at path '%s'", opts.TransformationPath))
	resp, err := client.UpdateTransformation(
		client.NewApiUpdateTransformationRequest(
			packageJson.TransformationID,
			ingestion.NewTransformationCreate(string(code), packageJson.Name).
				SetDescription("Transformation created from the Algolia CLI tool"),
		))
	if err != nil {
		return err
	}

	// and create a new global one, since they are immutable
	respGlobal, err := client.CreateTransformation(
		client.NewApiCreateTransformationRequest(
			ingestion.NewTransformationCreate(string(code), packageJson.Name).
				SetDescription("Transformation created from the Algolia CLI tool"),
		), ingestion.WithHeaderParam("Authorization", basicAuth))
	if err != nil {
		return err
	}

	err = transformationpackagetemplate.RefreshPackageJson(packageJson.Name, resp.GetTransformationID(), respGlobal.GetTransformationID())
	if err != nil {
		return fmt.Errorf("unable to refresh package.json: %w", err)
	}

	opts.IO.StopProgressIndicator()

	return nil
}
