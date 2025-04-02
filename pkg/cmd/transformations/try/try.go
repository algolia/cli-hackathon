package try

import (
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
	"github.com/algolia/cli/pkg/printers"
	"github.com/algolia/cli/pkg/validators"
)

type TryOptions struct {
	Config config.IConfig
	IO     *iostreams.IOStreams

	IngestionClient func() (*ingestion.APIClient, error)

	PrintFlags *cmdutil.PrintFlags
}

// NewTryCmd creates and returns a try command for transformations
func NewTryCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &TryOptions{
		IO:              f.IOStreams,
		Config:          f.Config,
		IngestionClient: f.IngestionClient,
		PrintFlags:      cmdutil.NewPrintFlags(),
	}
	cmd := &cobra.Command{
		Use:   "try",
		Args:  validators.NoArgs(),
		Short: "Try transformations",
		Example: heredoc.Doc(`
			# Try transformations
			$ algolia transfo try
			$ algolia transfo --local
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTryCmd(opts)
		},
		Annotations: map[string]string{
			"runInWebCLI": "true",
			"acls":        "addObject,deleteObject,listIndexes,deleteIndex,settings,editSettings",
		},
	}

	opts.PrintFlags.AddFlags(cmd)

	return cmd
}

func runTryCmd(opts *TryOptions) error {
	client, err := opts.IngestionClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicatorWithLabel("Trying transformation")

	res, err := client.ListTransformations(client.NewApiListTransformationsRequest())
	opts.IO.StopProgressIndicator()
	if err != nil {
		return err
	}

	if opts.PrintFlags.OutputFlagSpecified() && opts.PrintFlags.OutputFormat != nil {
		p, err := opts.PrintFlags.ToPrinter()
		if err != nil {
			return err
		}
		return p.Print(opts.IO, res)
	}

	if err := opts.IO.StartPager(); err != nil {
		fmt.Fprintf(opts.IO.ErrOut, "error starting pager: %v\n", err)
	}
	defer opts.IO.StopPager()

	table := printers.NewTablePrinter(opts.IO)
	if table.IsTTY() {
		table.AddField("NAME", nil, nil)
		table.AddField("DESCRIPTION", nil, nil)
		table.AddField("UPDATED AT", nil, nil)
		table.AddField("CREATED AT", nil, nil)
		table.EndRow()
	}

	for _, transfo := range res.Transformations {
		updatedAt, err := parseTime(*transfo.UpdatedAt)
		if err != nil {
			return fmt.Errorf("can't parse %s into a time struct", *transfo.UpdatedAt)
		}
		createdAt, err := parseTime(transfo.CreatedAt)
		if err != nil {
			return fmt.Errorf("can't parse %s into a time struct", transfo.CreatedAt)
		}

		desc := "None"
		if transfo.Description != nil {
			desc = *transfo.Description
		}

		table.AddField(transfo.Name, nil, nil)
		table.AddField(desc, nil, nil)
		table.AddField(updatedAt, nil, nil)
		table.AddField(createdAt, nil, nil)
		table.EndRow()
	}
	return table.Render()
}

// parseTime parses the string from the API response into a relative time string
func parseTime(timeAsString string) (string, error) {
	const layout = "2006-01-02T15:04:05.999Z"

	// This *should* restore the previous behavior when UpdatedAt is empty
	if timeAsString == "" {
		return "a long while ago", nil
	}

	t, err := time.Parse(layout, timeAsString)
	if err != nil {
		return "", err
	}

	return humanize.Time(t), nil
}
