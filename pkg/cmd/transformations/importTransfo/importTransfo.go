package importTransfo

import (
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	bubblelist "github.com/algolia/cli/pkg/cmd/transformations/bubble/list"
	"github.com/algolia/cli/pkg/cmd/transformations/setup/sourcepicker"
	"github.com/algolia/cli/pkg/cmd/transformations/setup/transformationpackagetemplate"
	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
	"github.com/algolia/cli/pkg/validators"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"os"
	"time"
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
			return runImportCmd(opts)
		},
		Annotations: map[string]string{
			"runInWebCLI": "true",
			"acls":        "addObject,deleteObject,listIndexes,deleteIndex,settings,editSettings",
		},
	}

	opts.PrintFlags.AddFlags(cmd)

	return cmd
}

func runImportCmd(opts *ImportOptions) error {
	client, err := opts.IngestionClient()
	if err != nil {
		return err
	}

	if opts.TransformationID == "" {
		opts.TransformationID, err = PickTransformation(client)
		if err != nil {
			return err
		}
	}

	opts.IO.StartProgressIndicatorWithLabel("Fetching transformation")
	res, err := client.GetTransformation(client.NewApiGetTransformationRequest(opts.TransformationID))
	opts.IO.StopProgressIndicator()
	if err != nil {
		return err
	}

	sourceID, err := sourcepicker.PickSource(client)
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicatorWithLabel(fmt.Sprintf("Sampling source %s", sourceID))

	resp, err := client.ValidateSourceBeforeUpdate(client.NewApiValidateSourceBeforeUpdateRequest(sourceID, ingestion.NewEmptySourceUpdate()))
	if err != nil {
		return err
	}

	outputDirectory := fmt.Sprintf("output%c%s", os.PathSeparator, res.Name)

	opts.IO.StartProgressIndicatorWithLabel(fmt.Sprintf("Generating output package folder at path '%s'", outputDirectory))

	if err = os.MkdirAll(outputDirectory, 0o750); err != nil {
		return fmt.Errorf("unable to create transformation folder with name '%s': %w", outputDirectory, err)
	}

	if err = transformationpackagetemplate.Generate(transformationpackagetemplate.PackageTemplate{
		TransformationName: res.Name,
		Sample:             resp.GetData()[0],
		Code:               &res.Code,
		TransformationID:   opts.TransformationID,
	}); err != nil {
		return err
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
		parse, _ := time.Parse(time.RFC3339, *transformation.UpdatedAt)
		items = append(items, bubblelist.Item{Name: fmt.Sprintf("%s (%s) - %s", transformation.GetName(), func() string {
			if transformation.Description != nil {
				return *transformation.Description
			}
			return "No description"
		}(), parse.String()), UUID: transformation.GetTransformationID()})
	}

	list := bubblelist.NewBubbleList("transformations", items)

	if _, err := tea.NewProgram(&list).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return list.Choice, nil
}
