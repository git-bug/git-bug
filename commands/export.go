package commands

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"

	"github.com/git-bug/git-bug/commands/execenv"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
	"github.com/git-bug/git-bug/query"
)

type exportOptions struct {
	actorQuery       []string
	authorQuery      []string
	labelQuery       []string
	noQuery          []string
	participantQuery []string
	queryOptions     []string
	statusQuery      []string
	titleQuery       []string
	sortBy           string
	sortDirection    string
}

func newExportCommand(env *execenv.Env) *cobra.Command {
	options := exportOptions{}

	cmd := &cobra.Command{
		Use:   "export [QUERY]",
		Short: "Export bugs as a series of operations.",
		Long: `Export bugs as a series of operations.

The output format is NDJSON (Newline delimited JSON).

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language, flags, a natural language full text search, or a combination of the aforementioned.`,
		Example: `Export all bug operations:
		git bug export
		Export all bug operations for a given participant:
		git bug export -p <name>
		`,
		PreRunE: execenv.LoadBackend(env),
		PostRunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			return runExport(env, options, args)
		}),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringSliceVarP(&options.statusQuery, "status", "s", nil,
		"Filter by status. Valid values are [open,closed]")
	flags.StringSliceVarP(&options.authorQuery, "author", "a", nil,
		"Filter by author")
	flags.StringSliceVarP(&options.participantQuery, "participant", "p", nil,
		"Filter by participant")
	flags.StringSliceVarP(&options.actorQuery, "actor", "A", nil,
		"Filter by actor")
	flags.StringSliceVarP(&options.labelQuery, "label", "l", nil,
		"Filter by label")
	flags.StringSliceVarP(&options.titleQuery, "title", "t", nil,
		"Filter by title")
	flags.StringSliceVarP(&options.noQuery, "no", "n", nil,
		"Filter by absence of something. Valid values are [label]")
	flags.StringVarP(&options.sortBy, "by", "b", "creation",
		"Sort the results by a characteristic. Valid values are [id,creation,edit]")
	flags.StringVarP(&options.sortDirection, "direction", "d", "asc",
		"Select the sorting direction. Valid values are [asc,desc]")

	return cmd
}

func runExport(env *execenv.Env, opts exportOptions, args []string) error {
	var q *query.Query
	var err error

	if len(args) >= 1 {
		q, err = query.Parse(strings.Join(args, " "))
		if err != nil {
			return err
		}
	} else {
		q = query.NewQuery()
	}

	// FIXME we are throwing away opts!
	allIds, err := env.Backend.Bugs().Query(q)
	if err != nil {
		return err
	}

	out := json.NewEncoder(env.Out)

	for _, id := range allIds {
		b, err := env.Backend.Bugs().Resolve(id)
		if err != nil {
			return err
		}

		wrapper := struct {
			Id         entity.Id       `json:"id"`
			Operations []dag.Operation `json:"operations"`
		}{
			Id:         id,
			Operations: b.Snapshot().Operations,
		}

		err = out.Encode(wrapper)
		if err != nil {
			return err
		}
	}

	return nil
}
