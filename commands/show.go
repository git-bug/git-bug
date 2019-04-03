package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
)

var (
	showFieldsQuery string
)

func runShowBug(cmd *cobra.Command, args []string) error {
	backend, err := cache.NewRepoCache(repo)
	if err != nil {
		return err
	}
	defer backend.Close()
	interrupt.RegisterCleaner(backend.Close)

	b, args, err := _select.ResolveBug(backend, args)
	if err != nil {
		return err
	}

	snapshot := b.Snapshot()

	if len(snapshot.Comments) == 0 {
		return errors.New("invalid bug: no comment")
	}

	firstComment := snapshot.Comments[0]

	if showFieldsQuery != "" {
		switch showFieldsQuery {
		case "author":
			fmt.Printf("%s\n", firstComment.Author.DisplayName())
		case "authorEmail":
			fmt.Printf("%s\n", firstComment.Author.Email())
		case "createTime":
			fmt.Printf("%s\n", firstComment.FormatTime())
		case "humanId":
			fmt.Printf("%s\n", snapshot.HumanId())
		case "id":
			fmt.Printf("%s\n", snapshot.Id())
		case "labels":
			var labels = make([]string, len(snapshot.Labels))
			for i, l := range snapshot.Labels {
				labels[i] = string(l)
			}
			fmt.Printf("%s\n", strings.Join(labels, "\n"))
		case "shortId":
			fmt.Printf("%s\n", snapshot.HumanId())
		case "status":
			fmt.Printf("%s\n", snapshot.Status)
		case "title":
			fmt.Printf("%s\n", snapshot.Title)
		default:
			return fmt.Errorf("\nUnsupported field: %s\n", showFieldsQuery)
		}

		return nil
	}

	// Header
	fmt.Printf("[%s] %s %s\n\n",
		colors.Yellow(snapshot.Status),
		colors.Cyan(snapshot.HumanId()),
		snapshot.Title,
	)

	fmt.Printf("%s opened this issue %s\n\n",
		colors.Magenta(firstComment.Author.DisplayName()),
		firstComment.FormatTimeRel(),
	)

	var labels = make([]string, len(snapshot.Labels))
	for i := range snapshot.Labels {
		labels[i] = string(snapshot.Labels[i])
	}

	fmt.Printf("labels: %s\n\n",
		strings.Join(labels, ", "),
	)

	// Comments
	indent := "  "

	for i, comment := range snapshot.Comments {
		var message string
		fmt.Printf("%s#%d %s <%s>\n\n",
			indent,
			i,
			comment.Author.DisplayName(),
			comment.Author.Email(),
		)

		if comment.Message == "" {
			message = colors.GreyBold("No description provided.")
		} else {
			message = comment.Message
		}

		fmt.Printf("%s%s\n\n\n",
			indent,
			message,
		)
	}

	return nil
}

var showCmd = &cobra.Command{
	Use:     "show [<id>]",
	Short:   "Display the details of a bug.",
	PreRunE: loadRepo,
	RunE:    runShowBug,
}

func init() {
	RootCmd.AddCommand(showCmd)
	showCmd.Flags().StringVarP(&showFieldsQuery, "field", "f", "",
		"Select field to display. Valid values are [author,authorEmail,createTime,humanId,id,labels,shortId,status,title]")
}
