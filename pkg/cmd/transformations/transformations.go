package transformations

import (
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmd/transformations/list"
	"github.com/algolia/cli/pkg/cmd/transformations/setup"
	"github.com/algolia/cli/pkg/cmdutil"
)

// NewTransformationsCmd returns a new command for transformations.
func NewTransformationsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "transformations",
		Aliases: []string{"transformations", "transfo"},
		Short:   "Manage your Algolia transformations",
	}

	cmd.AddCommand(list.NewListCmd(f))
	cmd.AddCommand(setup.NewSetupCmd(f))

	return cmd
}
