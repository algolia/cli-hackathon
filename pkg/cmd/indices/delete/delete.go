package delete

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/algolia/algoliasearch-client-go/v4/algolia/search"
	"github.com/spf13/cobra"

	"github.com/algolia/cli/pkg/cmdutil"
	"github.com/algolia/cli/pkg/config"
	"github.com/algolia/cli/pkg/iostreams"
	"github.com/algolia/cli/pkg/prompt"
)

type DeleteOptions struct {
	Config config.IConfig
	IO     *iostreams.IOStreams

	SearchClient func() (*search.APIClient, error)

	Indices         []string
	DoConfirm       bool
	IncludeReplicas bool
	Wait            bool
}

// NewDeleteCmd creates and returns a delete command for indices
func NewDeleteCmd(f *cmdutil.Factory, runF func(*DeleteOptions) error) *cobra.Command {
	opts := &DeleteOptions{
		IO:           f.IOStreams,
		Config:       f.Config,
		SearchClient: f.SearchClient,
	}

	var confirm bool

	cmd := &cobra.Command{
		Use:               "delete <index>",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: cmdutil.IndexNames(opts.SearchClient),
		Annotations: map[string]string{
			"acls": "deleteIndex",
		},
		Short: "Deletes the specified index and all its settings.",
		Long: heredoc.Doc(`
			Delete an index.
			Deleting an index does not delete its analytics data.
			If you try to delete a non-existing index, the operation is ignored without warning.
			If the index you want to delete has replica indices, the replicas become independent indices.
			If the index you want to delete is a replica index, you must first unlink it from its primary index before you can delete it.
		`),
		Example: heredoc.Doc(`
			# Delete the index named "MOVIES"
			$ algolia indices delete MOVIES

      # Delete the index named "MOVIES" and its replicas
      $ algolia indices delete MOVIES --include-replicas

			# Delete the index named "MOVIES", skipping the confirmation prompt
			$ algolia indices delete MOVIES -y

			# Delete multiple indices
			$ algolia indices delete MOVIES SERIES ANIMES
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Indices = args

			if !confirm {
				if !opts.IO.CanPrompt() {
					return cmdutil.FlagErrorf(
						"--confirm required when non-interactive shell is detected",
					)
				}
				opts.DoConfirm = true
			}

			if runF != nil {
				return runF(opts)
			}

			return runDeleteCmd(opts)
		},
	}

	cmd.Flags().
		BoolVarP(&confirm, "confirm", "y", false, "Skip the delete index confirmation prompt")
	cmd.Flags().
		BoolVarP(&opts.IncludeReplicas, "include-replicas", "r", false, "delete replica indices too")
	cmd.Flags().BoolVarP(&opts.Wait, "wait", "w", false, "wait for the operation to complete")

	return cmd
}

func runDeleteCmd(opts *DeleteOptions) error {
	client, err := opts.SearchClient()
	if err != nil {
		return err
	}

	// For nicer output
	indexSingularOrPlural := "index"
	if len(opts.Indices) > 1 {
		indexSingularOrPlural = "indices"
	}

	if opts.DoConfirm {
		var confirmed bool
		msg := fmt.Sprintf(
			"Are you sure you want to delete the %s %q?",
			indexSingularOrPlural,
			strings.Join(opts.Indices, ", "),
		)
		if opts.IncludeReplicas {
			msg = fmt.Sprintf(
				"Are you sure you want to delete the %s %q including their replicas?",
				indexSingularOrPlural,
				strings.Join(opts.Indices, ", "),
			)
		}
		err := prompt.Confirm(msg, &confirmed)
		if err != nil {
			return fmt.Errorf("failed to prompt: %w", err)
		}
		if !confirmed {
			return nil
		}
	}

	for _, index := range opts.Indices {
		// Equivalent to `client.IndexExists` but provides settings already
		settings, err := client.GetSettings(client.NewApiGetSettingsRequest(index))
		if err != nil {
			return fmt.Errorf("can't get settings of index %s: %w", index, err)
		}

		// If both primary and replica are going to be deleted, we have to wait
		// Or the `SetSettings` call in `detachReplica` creates a new, empty index
		if settings.HasReplicas() {
			for _, r := range settings.Replicas {
				if contains(opts.Indices, r) {
					opts.Wait = true
				}
			}
		}

		// `index` is a replica index
		if settings.HasPrimary() {
			opts.IO.StartProgressIndicatorWithLabel(
				fmt.Sprintf("Detaching replica index %s from its primary", index),
			)
			err = detachReplica(index, *settings.Primary, client)
			if err != nil {
				opts.IO.StopProgressIndicator()
				return fmt.Errorf("can't detach index %s: %w", index, err)
			}
			opts.IO.StopProgressIndicator()
		}

		opts.IO.StartProgressIndicatorWithLabel(
			fmt.Sprintf("Deleting index %s", index),
		)
		res, err := client.DeleteIndex(client.NewApiDeleteIndexRequest(index))
		if err != nil {
			opts.IO.StopProgressIndicator()
			return fmt.Errorf("can't delete index %s: %w", index, err)
		}

		if !opts.IncludeReplicas && opts.Wait {
			opts.IO.UpdateProgressIndicatorLabel("Waiting for the task to complete")
			_, err := client.WaitForTask(index, res.TaskID)
			if err != nil {
				opts.IO.StopProgressIndicator()
				return err
			}
		}

		if opts.IncludeReplicas && len(settings.Replicas) > 0 {
			// Wait for primary to be deleted, otherwise deleting replicas might fail
			opts.IO.UpdateProgressIndicatorLabel("Waiting for the primary index to be deleted")
			_, err := client.WaitForTask(index, res.TaskID)
			if err != nil {
				opts.IO.StopProgressIndicator()
				return fmt.Errorf("error while waiting for index %s to be deleted: %w", index, err)
			}

			for _, replica := range settings.Replicas {
				// Virtual replicas have name `virtual(replica)`...
				pattern := regexp.MustCompile(`^virtual\((.*)\)$`)
				matches := pattern.FindStringSubmatch(replica)
				if len(matches) > 1 {
					// But when deleting, we need the bare name
					replica = matches[1]
					// For printing the summary
					opts.Indices = append(opts.Indices, replica)
				}

				opts.IO.UpdateProgressIndicatorLabel(
					fmt.Sprintf("Deleting replica %s", index),
				)
				res, err = client.DeleteIndex(client.NewApiDeleteIndexRequest(replica))
				if err != nil {
					opts.IO.StopProgressIndicator()
					return fmt.Errorf("can't delete replica %s: %w", replica, err)
				}
				if opts.Wait {
					_, err := client.WaitForTask(replica, res.TaskID)
					if err != nil {
						opts.IO.StopProgressIndicator()
						return err
					}
				}
			}
		}
		opts.IO.StopProgressIndicator()
	}

	cs := opts.IO.ColorScheme()
	if opts.IO.IsStdoutTTY() {
		fmt.Fprintf(
			opts.IO.Out,
			"%s Deleted %s %s\n",
			cs.SuccessIcon(),
			indexSingularOrPlural,
			strings.Join(opts.Indices, ", "),
		)
	}

	return nil
}

// Remove replica from `replicas` settings of the primary index
func detachReplica(replica string, primary string, client *search.APIClient) error {
	settings, err := client.GetSettings(client.NewApiGetSettingsRequest(primary))
	if err != nil {
		return fmt.Errorf("can't get settings of primary index %s: %w", primary, err)
	}

	if isVirtual(settings.Replicas, replica) {
		replica = fmt.Sprintf("virtual(%s)", replica)
	}

	newReplicas := removeReplica(settings.Replicas, replica)

	res, err := client.SetSettings(
		client.NewApiSetSettingsRequest(
			primary,
			search.NewIndexSettings().SetReplicas(newReplicas),
		),
	)
	if err != nil {
		return fmt.Errorf("can't detach replica %s from its primary %s: %w", replica, primary, err)
	}

	_, err = client.WaitForTask(primary, res.TaskID)
	if err != nil {
		return fmt.Errorf("can't wait for updating the primary's settings: %w", err)
	}

	return nil
}

// isVirtual checks whether an index is a virtual replica
func isVirtual(replicas []string, name string) bool {
	pattern := regexp.MustCompile(fmt.Sprintf(`^virtual\(%s\)$`, name))

	for _, i := range replicas {
		matches := pattern.MatchString(i)
		if matches {
			return true
		}
	}

	return false
}

// removeReplica returns a new slice without a replica
func removeReplica(replicas []string, name string) []string {
	for i, v := range replicas {
		if v == name {
			// Return a new slice without the given replica
			return append(replicas[:i], replicas[i+1:]...)
		}
	}
	return replicas
}

// contains checks if ele is in arr
func contains[T comparable](arr []T, ele T) bool {
	for _, i := range arr {
		if ele == i {
			return true
		}
	}
	return false
}
