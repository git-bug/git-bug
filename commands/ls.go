package commands

import (
	"fmt"
	"strings"

	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/query/ast"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

var (
	lsStatusQuery      []string
	lsAuthorQuery      []string
	lsParticipantQuery []string
	lsLabelQuery       []string
	lsTitleQuery       []string
	lsActorQuery       []string
	lsNoQuery          []string
	lsSortBy           string
	lsSortDirection    string
)

func runLsBug(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	var q *ast.Query
	if len(args) >= 1 {
		q, err = query.Parse(strings.Join(args, " "))

		if err != nil {
			return err
		}
	} else {
		q, err = lsQueryFromFlags()
		if err != nil {
			return err
		}
	}

	allIds := backend.QueryBugs(q)

	for _, id := range allIds {
		b, err := backend.ResolveBugExcerpt(id)
		if err != nil {
			return err
		}

		var name string
		if b.AuthorId != "" {
			author, err := backend.ResolveIdentityExcerpt(b.AuthorId)
			if err != nil {
				name = "<missing author data>"
			} else {
				name = author.DisplayName()
			}
		} else {
			name = b.LegacyAuthor.DisplayName()
		}

		var labelsTxt strings.Builder
		for _, l := range b.Labels {
			lc256 := l.Color().Term256()
			labelsTxt.WriteString(lc256.Escape())
			labelsTxt.WriteString(" ◼")
			labelsTxt.WriteString(lc256.Unescape())
		}

		// truncate + pad if needed
		labelsFmt := text.TruncateMax(labelsTxt.String(), 10)
		titleFmt := text.LeftPadMaxLine(b.Title, 50-text.Len(labelsFmt), 0)
		authorFmt := text.LeftPadMaxLine(name, 15, 0)

		comments := fmt.Sprintf("%4d 💬", b.LenComments)
		if b.LenComments > 9999 {
			comments = "    ∞ 💬"
		}

		fmt.Printf("%s %s\t%s\t%s\t%s\n",
			colors.Cyan(b.Id.Human()),
			colors.Yellow(b.Status),
			titleFmt+labelsFmt,
			colors.Magenta(authorFmt),
			comments,
		)
	}

	return nil
}

// Transform the command flags into an ast.Query
func lsQueryFromFlags() (*ast.Query, error) {
	q := ast.NewQuery()

	for _, str := range lsStatusQuery {
		status, err := bug.StatusFromString(str)
		if err != nil {
			return nil, err
		}
		q.Status = append(q.Status, status)
	}
	for _, title := range lsTitleQuery {
		q.Title = append(q.Title, title)
	}
	for _, author := range lsAuthorQuery {
		q.Author = append(q.Author, author)
	}
	for _, actor := range lsActorQuery {
		q.Actor = append(q.Actor, actor)
	}
	for _, participant := range lsParticipantQuery {
		q.Participant = append(q.Participant, participant)
	}
	for _, label := range lsLabelQuery {
		q.Label = append(q.Label, label)
	}

	for _, no := range lsNoQuery {
		switch no {
		case "label":
			q.NoLabel = true
		default:
			return nil, fmt.Errorf("unknown \"no\" filter %s", no)
		}
	}

	switch lsSortBy {
	case "id":
		q.OrderBy = ast.OrderById
	case "creation":
		q.OrderBy = ast.OrderByCreation
	case "edit":
		q.OrderBy = ast.OrderByEdit
	default:
		return nil, fmt.Errorf("unknown sort flag %s", lsSortBy)
	}

	switch lsSortDirection {
	case "asc":
		q.OrderDirection = ast.OrderAscending
	case "desc":
		q.OrderDirection = ast.OrderDescending
	default:
		return nil, fmt.Errorf("unknown sort direction %s", lsSortDirection)
	}

	return q, nil
}

var lsCmd = &cobra.Command{
	Use:   "ls [<query>]",
	Short: "List bugs.",
	Long: `Display a summary of each bugs.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language or with flags.`,
	Example: `List open bugs sorted by last edition with a query:
git bug ls status:open sort:edit-desc

List closed bugs sorted by creation with flags:
git bug ls --status closed --by creation
`,
	PreRunE: loadRepo,
	RunE:    runLsBug,
}

func init() {
	RootCmd.AddCommand(lsCmd)

	lsCmd.Flags().SortFlags = false

	lsCmd.Flags().StringSliceVarP(&lsStatusQuery, "status", "s", nil,
		"Filter by status. Valid values are [open,closed]")
	lsCmd.Flags().StringSliceVarP(&lsAuthorQuery, "author", "a", nil,
		"Filter by author")
	lsCmd.Flags().StringSliceVarP(&lsParticipantQuery, "participant", "p", nil,
		"Filter by participant")
	lsCmd.Flags().StringSliceVarP(&lsActorQuery, "actor", "A", nil,
		"Filter by actor")
	lsCmd.Flags().StringSliceVarP(&lsLabelQuery, "label", "l", nil,
		"Filter by label")
	lsCmd.Flags().StringSliceVarP(&lsTitleQuery, "title", "t", nil,
		"Filter by title")
	lsCmd.Flags().StringSliceVarP(&lsNoQuery, "no", "n", nil,
		"Filter by absence of something. Valid values are [label]")
	lsCmd.Flags().StringVarP(&lsSortBy, "by", "b", "creation",
		"Sort the results by a characteristic. Valid values are [id,creation,edit]")
	lsCmd.Flags().StringVarP(&lsSortDirection, "direction", "d", "asc",
		"Select the sorting direction. Valid values are [asc,desc]")
}
