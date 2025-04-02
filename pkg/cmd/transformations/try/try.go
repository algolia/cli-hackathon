package try

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/ingestion"
	"github.com/algolia/cli/pkg/printers"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
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

	transformation, err := os.ReadFile("src/transformation.js")
	if err != nil {
		return fmt.Errorf("failed to read transformation file: %w", err)
	}

	sampleRaw, err := os.ReadFile("sample.json")
	if err != nil {
		return fmt.Errorf("failed to read sample file: %w", err)
	}

	var sample map[string]any
	if err = json.Unmarshal(sampleRaw, &sample); err != nil {
		return fmt.Errorf("failed to unmarshal sample file, make sure it's a valid JSON object: %w", err)
	}

	res, body, err := client.TryTransformationWithHTTPInfo(client.NewApiTryTransformationRequest(ingestion.NewTransformationTry(string(transformation), sample)))
	opts.IO.StopProgressIndicator()
	if err != nil {
		return err
	}

	var result struct {
		Payloads []string                       `json:"payloads"`
		Error    *ingestion.TransformationError `json:"error,omitempty"`
	}

	if err = json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("failed to try transformation: %s", *result.Error.Message)
	}

	var firstPayload map[string]any
	if err = json.Unmarshal([]byte(result.Payloads[0]), &firstPayload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return (&printers.JSONPrinter{}).Print(opts.IO, firstPayload)

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
