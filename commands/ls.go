package commands

import (
	"fmt"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util"
	"github.com/spf13/cobra"
)

var (
	lsStatusQuery   []string
	lsAuthorQuery   []string
	lsLabelQuery    []string
	lsNoQuery       []string
	lsSortBy        string
	lsSortDirection string
)

func runLsBug(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()

	var query *cache.Query
	if len(args) >= 1 {
		query, err = cache.ParseQuery(args[0])

		if err != nil {
			return err
		}
	} else {
		query, err = lsQueryFromFlags()
		if err != nil {
			return err
		}
	}

	allIds := backend.QueryBugs(query)

	for _, id := range allIds {
		b, err := backend.ResolveBug(id)
		if err != nil {
			return err
		}

		snapshot := b.Snapshot()

		var author bug.Person

		if len(snapshot.Comments) > 0 {
			create := snapshot.Comments[0]
			author = create.Author
		}

		// truncate + pad if needed
		titleFmt := fmt.Sprintf("%-50.50s", snapshot.Title)
		authorFmt := fmt.Sprintf("%-15.15s", author.Name)

		fmt.Printf("%s %s\t%s\t%s\t%s\n",
			util.Cyan(b.HumanId()),
			util.Yellow(snapshot.Status),
			titleFmt,
			util.Magenta(authorFmt),
			snapshot.Summary(),
		)
	}

	return nil
}

// Transform the command flags into a query
func lsQueryFromFlags() (*cache.Query, error) {
	query := cache.NewQuery()

	for _, status := range lsStatusQuery {
		f, err := cache.StatusFilter(status)
		if err != nil {
			return nil, err
		}
		query.Status = append(query.Status, f)
	}

	for _, author := range lsAuthorQuery {
		f := cache.AuthorFilter(author)
		query.Author = append(query.Author, f)
	}

	for _, label := range lsLabelQuery {
		f := cache.LabelFilter(label)
		query.Label = append(query.Label, f)
	}

	for _, no := range lsNoQuery {
		switch no {
		case "label":
			query.NoFilters = append(query.NoFilters, cache.NoLabelFilter())
		default:
			return nil, fmt.Errorf("unknown \"no\" filter %s", no)
		}
	}

	switch lsSortBy {
	case "id":
		query.OrderBy = cache.OrderById
	case "creation":
		query.OrderBy = cache.OrderByCreation
	case "edit":
		query.OrderBy = cache.OrderByEdit
	default:
		return nil, fmt.Errorf("unknown sort flag %s", lsSortBy)
	}

	switch lsSortDirection {
	case "asc":
		query.OrderDirection = cache.OrderAscending
	case "desc":
		query.OrderDirection = cache.OrderDescending
	default:
		return nil, fmt.Errorf("unknown sort direction %s", lsSortDirection)
	}

	return query, nil
}

var lsCmd = &cobra.Command{
	Use:   "ls [<query>]",
	Short: "List bugs",
	Long: `Display a summary of each bugs.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language or with flags.`,
	Example: `List open bugs sorted by last edition with a query:
git bug ls "status:open sort:edit-desc"

List closed bugs sorted by creation with flags:
git bug ls --status closed --by creation
`,
	RunE: runLsBug,
}

func init() {
	RootCmd.AddCommand(lsCmd)

	lsCmd.Flags().SortFlags = false

	lsCmd.Flags().StringSliceVarP(&lsStatusQuery, "status", "s", nil,
		"Filter by status. Valid values are [open,closed]")
	lsCmd.Flags().StringSliceVarP(&lsAuthorQuery, "author", "a", nil,
		"Filter by author")
	lsCmd.Flags().StringSliceVarP(&lsLabelQuery, "label", "l", nil,
		"Filter by label")
	lsCmd.Flags().StringSliceVarP(&lsNoQuery, "no", "n", nil,
		"Filter by absence of something. Valid values are [label]")
	lsCmd.Flags().StringVarP(&lsSortBy, "by", "b", "creation",
		"Sort the results by a characteristic. Valid values are [id,creation,edit]")
	lsCmd.Flags().StringVarP(&lsSortDirection, "direction", "d", "asc",
		"Select the sorting direction. Valid values are [asc,desc]")
}
