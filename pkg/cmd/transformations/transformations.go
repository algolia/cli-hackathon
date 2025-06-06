package transformations

import (
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmd/transformations/importTransfo"
	"github.com/algolia/cli/pkg/cmd/transformations/link"
	"github.com/algolia/cli/pkg/cmd/transformations/list"
	"github.com/algolia/cli/pkg/cmd/transformations/sample"
	"github.com/algolia/cli/pkg/cmd/transformations/save"
	"github.com/algolia/cli/pkg/cmd/transformations/setup"
	"github.com/algolia/cli/pkg/cmd/transformations/try"
	"github.com/algolia/cli/pkg/cmdutil"
)

// NewTransformationsCmd returns a new command for transformations.
func NewTransformationsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transformations",
		Aliases: []string{"transformations", "transfo", "tr", "t"},
		Short:   "Manage your Algolia transformations",
	}

	cmd.AddCommand(list.NewListCmd(f))
	cmd.AddCommand(try.NewTryCmd(f))
	cmd.AddCommand(setup.NewSetupCmd(f))
	cmd.AddCommand(save.NewSaveCmd(f))
	cmd.AddCommand(importTransfo.NewImportCmd(f))
	cmd.AddCommand(sample.NewSampleCmd(f))
	cmd.AddCommand(link.NewLinkCmd(f))

	return cmd
}
