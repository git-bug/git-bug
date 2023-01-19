package bugcmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	text "github.com/MichaelMure/go-term-text"
	"github.com/spf13/cobra"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/cmdjson"
	"github.com/MichaelMure/git-bug/commands/completion"
	"github.com/MichaelMure/git-bug/commands/execenv"
	"github.com/MichaelMure/git-bug/entities/bug"
	"github.com/MichaelMure/git-bug/entities/common"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/query"
	"github.com/MichaelMure/git-bug/util/colors"
)

type bugOptions struct {
	statusQuery         []string
	authorQuery         []string
	metadataQuery       []string
	participantQuery    []string
	actorQuery          []string
	labelQuery          []string
	titleQuery          []string
	noQuery             []string
	sortBy              string
	sortDirection       string
	outputFormat        string
	outputFormatChanged bool
}

func NewBugCommand(env *execenv.Env) *cobra.Command {
	options := bugOptions{}

	cmd := &cobra.Command{
		Use:   "bug [QUERY]",
		Short: "List bugs",
		Long: `Display a summary of each bugs.

You can pass an additional query to filter and order the list. This query can be expressed either with a simple query language, flags, a natural language full text search, or a combination of the aforementioned.`,
		Example: `List open bugs sorted by last edition with a query:
git bug status:open sort:edit-desc

List closed bugs sorted by creation with flags:
git bug --status closed --by creation

Do a full text search of all bugs:
git bug "foo bar" baz

Use queries, flags, and full text search:
git bug status:open --by creation "foo bar" baz
`,
		PreRunE: execenv.LoadBackend(env),
		RunE: execenv.CloseBackend(env, func(cmd *cobra.Command, args []string) error {
			options.outputFormatChanged = cmd.Flags().Changed("format")
			return runBug(env, options, args)
		}),
		ValidArgsFunction: completion.Ls(env),
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringSliceVarP(&options.statusQuery, "status", "s", nil,
		"Filter by status. Valid values are [open,closed]")
	cmd.RegisterFlagCompletionFunc("status", completion.From([]string{"open", "closed"}))
	flags.StringSliceVarP(&options.authorQuery, "author", "a", nil,
		"Filter by author")
	flags.StringSliceVarP(&options.metadataQuery, "metadata", "m", nil,
		"Filter by metadata. Example: github-url=URL")
	cmd.RegisterFlagCompletionFunc("author", completion.UserForQuery(env))
	flags.StringSliceVarP(&options.participantQuery, "participant", "p", nil,
		"Filter by participant")
	cmd.RegisterFlagCompletionFunc("participant", completion.UserForQuery(env))
	flags.StringSliceVarP(&options.actorQuery, "actor", "A", nil,
		"Filter by actor")
	cmd.RegisterFlagCompletionFunc("actor", completion.UserForQuery(env))
	flags.StringSliceVarP(&options.labelQuery, "label", "l", nil,
		"Filter by label")
	cmd.RegisterFlagCompletionFunc("label", completion.Label(env))
	flags.StringSliceVarP(&options.titleQuery, "title", "t", nil,
		"Filter by title")
	flags.StringSliceVarP(&options.noQuery, "no", "n", nil,
		"Filter by absence of something. Valid values are [label]")
	cmd.RegisterFlagCompletionFunc("no", completion.Label(env))
	flags.StringVarP(&options.sortBy, "by", "b", "creation",
		"Sort the results by a characteristic. Valid values are [id,creation,edit]")
	cmd.RegisterFlagCompletionFunc("by", completion.From([]string{"id", "creation", "edit"}))
	flags.StringVarP(&options.sortDirection, "direction", "d", "asc",
		"Select the sorting direction. Valid values are [asc,desc]")
	cmd.RegisterFlagCompletionFunc("direction", completion.From([]string{"asc", "desc"}))
	flags.StringVarP(&options.outputFormat, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,plain,id,json,org-mode]")
	cmd.RegisterFlagCompletionFunc("format",
		completion.From([]string{"default", "plain", "id", "json", "org-mode"}))

	const selectGroup = "select"
	cmd.AddGroup(&cobra.Group{ID: selectGroup, Title: "Implicit selection"})

	addCmdWithGroup := func(child *cobra.Command, groupID string) {
		cmd.AddCommand(child)
		child.GroupID = groupID
	}

	addCmdWithGroup(newBugDeselectCommand(env), selectGroup)
	addCmdWithGroup(newBugSelectCommand(env), selectGroup)

	cmd.AddCommand(newBugCommentCommand(env))
	cmd.AddCommand(newBugLabelCommand(env))
	cmd.AddCommand(newBugNewCommand(env))
	cmd.AddCommand(newBugRmCommand(env))
	cmd.AddCommand(newBugShowCommand(env))
	cmd.AddCommand(newBugStatusCommand(env))
	cmd.AddCommand(newBugTitleCommand(env))

	return cmd
}

func runBug(env *execenv.Env, opts bugOptions, args []string) error {
	var q *query.Query
	var err error

	if len(args) >= 1 {
		// either the shell or cobra remove the quotes, we need them back for the query parsing
		assembled := repairQuery(args)

		q, err = query.Parse(assembled)
		if err != nil {
			return err
		}
	} else {
		q = query.NewQuery()
	}

	err = completeQuery(q, opts)
	if err != nil {
		return err
	}

	allIds, err := env.Backend.Bugs().Query(q)
	if err != nil {
		return err
	}

	excerpts := make([]*cache.BugExcerpt, len(allIds))
	for i, id := range allIds {
		b, err := env.Backend.Bugs().ResolveExcerpt(id)
		if err != nil {
			return err
		}
		excerpts[i] = b
	}

	switch opts.outputFormat {
	case "default":
		if opts.outputFormatChanged {
			return bugsDefaultFormatter(env, excerpts)
		}
		if env.Out.IsTerminal() {
			return bugsDefaultFormatter(env, excerpts)
		} else {
			return bugsPlainFormatter(env, excerpts)
		}
	case "id":
		return bugsIDFormatter(env, excerpts)
	case "plain":
		return bugsPlainFormatter(env, excerpts)
	case "json":
		return bugsJsonFormatter(env, excerpts)
	case "org-mode":
		return bugsOrgmodeFormatter(env, excerpts)
	default:
		return fmt.Errorf("unknown format %s", opts.outputFormat)
	}
}

func repairQuery(args []string) string {
	for i, arg := range args {
		split := strings.Split(arg, ":")
		for j, s := range split {
			if strings.Contains(s, " ") {
				split[j] = fmt.Sprintf("\"%s\"", s)
			}
		}
		args[i] = strings.Join(split, ":")
	}
	return strings.Join(args, " ")
}

func bugsJsonFormatter(env *execenv.Env, excerpts []*cache.BugExcerpt) error {
	jsonBugs := make([]cmdjson.BugExcerpt, len(excerpts))
	for i, b := range excerpts {
		jsonBug, err := cmdjson.NewBugExcerpt(env.Backend, b)
		if err != nil {
			return err
		}
		jsonBugs[i] = jsonBug
	}
	return env.Out.PrintJSON(jsonBugs)
}

func bugsIDFormatter(env *execenv.Env, excerpts []*cache.BugExcerpt) error {
	for _, b := range excerpts {
		env.Out.Println(b.Id().String())
	}

	return nil
}

func bugsDefaultFormatter(env *execenv.Env, excerpts []*cache.BugExcerpt) error {
	width := env.Out.Width()
	widthId := entity.HumanIdLength
	widthStatus := len("closed")
	widthComment := 6

	widthRemaining := width -
		widthId - 1 -
		widthStatus - 1 -
		widthComment - 1

	widthTitle := int(float32(widthRemaining-3) * 0.7)
	if widthTitle < 0 {
		widthTitle = 0
	}

	widthRemaining = widthRemaining - widthTitle - 3 - 2
	widthAuthor := widthRemaining

	for _, b := range excerpts {
		author, err := env.Backend.Identities().ResolveExcerpt(b.AuthorId)
		if err != nil {
			return err
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
		titleFmt := text.LeftPadMaxLine(strings.TrimSpace(b.Title), widthTitle-text.Len(labelsFmt), 0)
		authorFmt := text.LeftPadMaxLine(author.DisplayName(), widthAuthor, 0)

		comments := fmt.Sprintf("%3d 💬", b.LenComments-1)
		if b.LenComments-1 <= 0 {
			comments = ""
		}
		if b.LenComments-1 > 999 {
			comments = "  ∞ 💬"
		}

		env.Out.Printf("%s\t%s\t%s   %s %s\n",
			colors.Cyan(b.Id().Human()),
			colors.Yellow(b.Status),
			titleFmt+labelsFmt,
			colors.Magenta(authorFmt),
			comments,
		)
	}
	return nil
}

func bugsPlainFormatter(env *execenv.Env, excerpts []*cache.BugExcerpt) error {
	for _, b := range excerpts {
		env.Out.Printf("%s\t%s\t%s\n", b.Id().Human(), b.Status, strings.TrimSpace(b.Title))
	}
	return nil
}

func bugsOrgmodeFormatter(env *execenv.Env, excerpts []*cache.BugExcerpt) error {
	// see https://orgmode.org/manual/Tags.html
	orgTagRe := regexp.MustCompile("[^[:alpha:]_@]")
	formatTag := func(l bug.Label) string {
		return orgTagRe.ReplaceAllString(l.String(), "_")
	}

	formatTime := func(time time.Time) string {
		return time.Format("[2006-01-02 Mon 15:05]")
	}

	env.Out.Println("#+TODO: OPEN | CLOSED")

	for _, b := range excerpts {
		status := strings.ToUpper(b.Status.String())

		var title string
		if link, ok := b.CreateMetadata["github-url"]; ok {
			title = fmt.Sprintf("[[%s][%s]]", link, b.Title)
		} else {
			title = b.Title
		}

		author, err := env.Backend.Identities().ResolveExcerpt(b.AuthorId)
		if err != nil {
			return err
		}

		var labels strings.Builder
		labels.WriteString(":")
		for i, l := range b.Labels {
			if i > 0 {
				labels.WriteString(":")
			}
			labels.WriteString(formatTag(l))
		}
		labels.WriteString(":")

		env.Out.Printf("* %-6s %s %s %s: %s %s\n",
			status,
			b.Id().Human(),
			formatTime(b.CreateTime()),
			author.DisplayName(),
			title,
			labels.String(),
		)

		env.Out.Printf("** Last Edited: %s\n", formatTime(b.EditTime()))

		env.Out.Printf("** Actors:\n")
		for _, element := range b.Actors {
			actor, err := env.Backend.Identities().ResolveExcerpt(element)
			if err != nil {
				return err
			}

			env.Out.Printf(": %s %s\n",
				actor.Id().Human(),
				actor.DisplayName(),
			)
		}

		env.Out.Printf("** Participants:\n")
		for _, element := range b.Participants {
			participant, err := env.Backend.Identities().ResolveExcerpt(element)
			if err != nil {
				return err
			}

			env.Out.Printf(": %s %s\n",
				participant.Id().Human(),
				participant.DisplayName(),
			)
		}
	}

	return nil
}

// Finish the command flags transformation into the query.Query
func completeQuery(q *query.Query, opts bugOptions) error {
	for _, str := range opts.statusQuery {
		status, err := common.StatusFromString(str)
		if err != nil {
			return err
		}
		q.Status = append(q.Status, status)
	}

	q.Author = append(q.Author, opts.authorQuery...)
	for _, str := range opts.metadataQuery {
		tokens := strings.Split(str, "=")
		if len(tokens) < 2 {
			return fmt.Errorf("no \"=\" in key=value metadata markup")
		}
		var pair query.StringPair
		pair.Key = tokens[0]
		pair.Value = tokens[1]
		q.Metadata = append(q.Metadata, pair)
	}
	q.Participant = append(q.Participant, opts.participantQuery...)
	q.Actor = append(q.Actor, opts.actorQuery...)
	q.Label = append(q.Label, opts.labelQuery...)
	q.Title = append(q.Title, opts.titleQuery...)

	for _, no := range opts.noQuery {
		switch no {
		case "label":
			q.NoLabel = true
		default:
			return fmt.Errorf("unknown \"no\" filter %s", no)
		}
	}

	switch opts.sortBy {
	case "id":
		q.OrderBy = query.OrderById
	case "creation":
		q.OrderBy = query.OrderByCreation
	case "edit":
		q.OrderBy = query.OrderByEdit
	default:
		return fmt.Errorf("unknown sort flag %s", opts.sortBy)
	}

	switch opts.sortDirection {
	case "asc":
		q.OrderDirection = query.OrderAscending
	case "desc":
		q.OrderDirection = query.OrderDescending
	default:
		return fmt.Errorf("unknown sort direction %s", opts.sortDirection)
	}

	return nil
}
