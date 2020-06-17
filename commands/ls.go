package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
)

var (
	lsQuery query.Query

	lsStatusQuery   []string
	lsNoQuery       []string
	lsSortBy        string
	lsSortDirection string
	lsOutputFormat  string
)

func runLsBug(_ *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	var q *query.Query
	if len(args) >= 1 {
		q, err = query.Parse(strings.Join(args, " "))

		if err != nil {
			return err
		}
	} else {
		err = completeQuery()
		if err != nil {
			return err
		}
		q = &lsQuery
	}

	allIds := backend.QueryBugs(q)

	bugExcerpt := make([]*cache.BugExcerpt, len(allIds))
	for i, id := range allIds {
		b, err := backend.ResolveBugExcerpt(id)
		if err != nil {
			return err
		}
		bugExcerpt[i] = b
	}

	switch lsOutputFormat {
	case "plain":
		return lsPlainFormatter(backend, bugExcerpt)
	case "json":
		return lsJsonFormatter(backend, bugExcerpt)
	case "default":
		return lsDefaultFormatter(backend, bugExcerpt)
	default:
		return fmt.Errorf("unknown format %s", lsOutputFormat)
	}
}

type JSONBug struct {
	Id           string    `json:"id"`
	HumanId      string    `json:"human_id"`
	CreationTime time.Time `json:"creation_time"`
	LastEdited   time.Time `json:"last_edited"`

	Status       string         `json:"status"`
	Labels       []bug.Label    `json:"labels"`
	Title        string         `json:"title"`
	Actors       []JSONIdentity `json:"actors"`
	Participants []JSONIdentity `json:"participants"`
	Author       JSONIdentity   `json:"author"`

	Comments int               `json:"comments"`
	Metadata map[string]string `json:"metadata"`
}

type JSONIdentity struct {
	Id      string `json:"id"`
	HumanId string `json:"human_id"`
	Name    string `json:"name"`
	Login   string `json:"login"`
}

func lsJsonFormatter(backend *cache.RepoCache, bugExcerpts []*cache.BugExcerpt) error {
	jsonBugs := make([]JSONBug, len(bugExcerpts))
	for i, b := range bugExcerpts {
		jsonBug := JSONBug{
			b.Id.String(),
			b.Id.Human(),
			time.Unix(b.CreateUnixTime, 0),
			time.Unix(b.EditUnixTime, 0),
			b.Status.String(),
			b.Labels,
			b.Title,
			[]JSONIdentity{},
			[]JSONIdentity{},
			JSONIdentity{},
			b.LenComments,
			b.CreateMetadata,
		}

		if b.AuthorId != "" {
			author, err := backend.ResolveIdentityExcerpt(b.AuthorId)
			if err != nil {
				return err
			}

			jsonBug.Author.Name = author.DisplayName()
			jsonBug.Author.Login = author.Login
			jsonBug.Author.Id = author.Id.String()
			jsonBug.Author.HumanId = author.Id.Human()
		} else {
			jsonBug.Author.Name = b.LegacyAuthor.DisplayName()
			jsonBug.Author.Login = b.LegacyAuthor.Login
		}

		for _, element := range b.Actors {
			actor, err := backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}

			jsonBug.Actors = append(jsonBug.Actors, JSONIdentity{
				actor.Id.String(),
				actor.Id.Human(),
				actor.Name,
				actor.Login,
			})
		}

		for _, element := range b.Participants {
			participant, err := backend.ResolveIdentityExcerpt(element)
			if err != nil {
				return err
			}
			jsonBug.Participants = append(jsonBug.Participants, JSONIdentity{
				participant.Id.String(),
				participant.Id.Human(),
				participant.DisplayName(),
				participant.Login,
			})
		}

		jsonBugs[i] = jsonBug
	}
	jsonObject, _ := json.MarshalIndent(jsonBugs, "", "    ")
	fmt.Printf("%s\n", jsonObject)
	return nil
}

func lsDefaultFormatter(backend *cache.RepoCache, bugExcerpts []*cache.BugExcerpt) error {
	for _, b := range bugExcerpts {
		var name string
		if b.AuthorId != "" {
			author, err := backend.ResolveIdentityExcerpt(b.AuthorId)
			if err != nil {
				return err
			}
			name = author.DisplayName()
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

func lsPlainFormatter(_ *cache.RepoCache, bugExcerpts []*cache.BugExcerpt) error {
	for _, b := range bugExcerpts {
		fmt.Printf("[%s] %s\n", b.Status, b.Title)
	}
	return nil
}

// Finish the command flags transformation into the query.Query
func completeQuery() error {
	for _, str := range lsStatusQuery {
		status, err := bug.StatusFromString(str)
		if err != nil {
			return err
		}
		lsQuery.Status = append(lsQuery.Status, status)
	}

	for _, no := range lsNoQuery {
		switch no {
		case "label":
			lsQuery.NoLabel = true
		default:
			return fmt.Errorf("unknown \"no\" filter %s", no)
		}
	}

	switch lsSortBy {
	case "id":
		lsQuery.OrderBy = query.OrderById
	case "creation":
		lsQuery.OrderBy = query.OrderByCreation
	case "edit":
		lsQuery.OrderBy = query.OrderByEdit
	default:
		return fmt.Errorf("unknown sort flag %s", lsSortBy)
	}

	switch lsSortDirection {
	case "asc":
		lsQuery.OrderDirection = query.OrderAscending
	case "desc":
		lsQuery.OrderDirection = query.OrderDescending
	default:
		return fmt.Errorf("unknown sort direction %s", lsSortDirection)
	}

	return nil
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
	lsCmd.Flags().StringSliceVarP(&lsQuery.Author, "author", "a", nil,
		"Filter by author")
	lsCmd.Flags().StringSliceVarP(&lsQuery.Participant, "participant", "p", nil,
		"Filter by participant")
	lsCmd.Flags().StringSliceVarP(&lsQuery.Actor, "actor", "A", nil,
		"Filter by actor")
	lsCmd.Flags().StringSliceVarP(&lsQuery.Label, "label", "l", nil,
		"Filter by label")
	lsCmd.Flags().StringSliceVarP(&lsQuery.Title, "title", "t", nil,
		"Filter by title")
	lsCmd.Flags().StringSliceVarP(&lsNoQuery, "no", "n", nil,
		"Filter by absence of something. Valid values are [label]")
	lsCmd.Flags().StringVarP(&lsSortBy, "by", "b", "creation",
		"Sort the results by a characteristic. Valid values are [id,creation,edit]")
	lsCmd.Flags().StringVarP(&lsSortDirection, "direction", "d", "asc",
		"Select the sorting direction. Valid values are [asc,desc]")
	lsCmd.Flags().StringVarP(&lsOutputFormat, "format", "f", "default",
		"Select the output formatting style. Valid values are [default, plain(text), json]")
}
