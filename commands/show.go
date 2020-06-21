package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	_select "github.com/MichaelMure/git-bug/commands/select"
	"github.com/MichaelMure/git-bug/util/colors"
	"github.com/MichaelMure/git-bug/util/interrupt"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

var (
	showFieldsQuery  string
	showOutputFormat string
)

func runShowBug(_ *cobra.Command, args []string) error {
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

	if showFieldsQuery != "" {
		switch showFieldsQuery {
		case "author":
			fmt.Printf("%s\n", snapshot.Author.DisplayName())
		case "authorEmail":
			fmt.Printf("%s\n", snapshot.Author.Email())
		case "createTime":
			fmt.Printf("%s\n", snapshot.CreatedAt.String())
		case "lastEdit":
			fmt.Printf("%s\n", snapshot.LastEditTime().String())
		case "humanId":
			fmt.Printf("%s\n", snapshot.Id().Human())
		case "id":
			fmt.Printf("%s\n", snapshot.Id())
		case "labels":
			for _, l := range snapshot.Labels {
				fmt.Printf("%s\n", l.String())
			}
		case "actors":
			for _, a := range snapshot.Actors {
				fmt.Printf("%s\n", a.DisplayName())
			}
		case "participants":
			for _, p := range snapshot.Participants {
				fmt.Printf("%s\n", p.DisplayName())
			}
		case "shortId":
			fmt.Printf("%s\n", snapshot.Id().Human())
		case "status":
			fmt.Printf("%s\n", snapshot.Status)
		case "title":
			fmt.Printf("%s\n", snapshot.Title)
		default:
			return fmt.Errorf("\nUnsupported field: %s\n", showFieldsQuery)
		}

		return nil
	}

	switch showOutputFormat {
	case "json":
		return showJsonFormatter(snapshot)
	case "plain":
		return showPlainFormatter(snapshot)
	case "default":
		return showDefaultFormatter(snapshot)
	default:
		return fmt.Errorf("unknown format %s", showOutputFormat)
	}
}

func showDefaultFormatter(snapshot *bug.Snapshot) error {
	// Header
	fmt.Printf("[%s] %s %s\n\n",
		colors.Yellow(snapshot.Status),
		colors.Cyan(snapshot.Id().Human()),
		snapshot.Title,
	)

	fmt.Printf("%s opened this issue %s\n",
		colors.Magenta(snapshot.Author.DisplayName()),
		snapshot.CreatedAt.String(),
	)

	fmt.Printf("This was last edited at %s\n\n",
		snapshot.LastEditTime().String(),
	)

	// Labels
	var labels = make([]string, len(snapshot.Labels))
	for i := range snapshot.Labels {
		labels[i] = string(snapshot.Labels[i])
	}

	fmt.Printf("labels: %s\n",
		strings.Join(labels, ", "),
	)

	// Actors
	var actors = make([]string, len(snapshot.Actors))
	for i := range snapshot.Actors {
		actors[i] = snapshot.Actors[i].DisplayName()
	}

	fmt.Printf("actors: %s\n",
		strings.Join(actors, ", "),
	)

	// Participants
	var participants = make([]string, len(snapshot.Participants))
	for i := range snapshot.Participants {
		participants[i] = snapshot.Participants[i].DisplayName()
	}

	fmt.Printf("participants: %s\n\n",
		strings.Join(participants, ", "),
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

func showPlainFormatter(snapshot *bug.Snapshot) error {
	// Header
	fmt.Printf("[%s] %s %s\n",
		snapshot.Status,
		snapshot.Id().Human(),
		snapshot.Title,
	)

	fmt.Printf("author: %s\n",
		snapshot.Author.DisplayName(),
	)

	fmt.Printf("creation time: %s\n",
		snapshot.CreatedAt.String(),
	)

	fmt.Printf("last edit: %s\n",
		snapshot.LastEditTime().String(),
	)

	// Labels
	var labels = make([]string, len(snapshot.Labels))
	for i := range snapshot.Labels {
		labels[i] = string(snapshot.Labels[i])
	}

	fmt.Printf("labels: %s\n",
		strings.Join(labels, ", "),
	)

	// Actors
	var actors = make([]string, len(snapshot.Actors))
	for i := range snapshot.Actors {
		actors[i] = snapshot.Actors[i].DisplayName()
	}

	fmt.Printf("actors: %s\n",
		strings.Join(actors, ", "),
	)

	// Participants
	var participants = make([]string, len(snapshot.Participants))
	for i := range snapshot.Participants {
		participants[i] = snapshot.Participants[i].DisplayName()
	}

	fmt.Printf("participants: %s\n",
		strings.Join(participants, ", "),
	)

	// Comments
	indent := "  "

	for i, comment := range snapshot.Comments {
		var message string
		fmt.Printf("%s#%d %s <%s>\n",
			indent,
			i,
			comment.Author.DisplayName(),
			comment.Author.Email(),
		)

		if comment.Message == "" {
			message = "No description provided."
		} else {
			message = comment.Message
		}

		fmt.Printf("%s%s\n",
			indent,
			message,
		)
	}

	return nil
}

type JSONBugSnapshot struct {
	Id           string    `json:"id"`
	HumanId      string    `json:"human_id"`
	CreationTime time.Time `json:"creation_time"`
	LastEdited   time.Time `json:"last_edited"`

	Status       string         `json:"status"`
	Labels       []bug.Label    `json:"labels"`
	Title        string         `json:"title"`
	Author       JSONIdentity   `json:"author"`
	Actors       []JSONIdentity `json:"actors"`
	Participants []JSONIdentity `json:"participants"`

	Comments []JSONComment `json:"comments"`
}

type JSONComment struct {
	Id          int    `json:"id"`
	AuthorName  string `json:"author_name"`
	AuthorLogin string `json:"author_login"`
	Message     string `json:"message"`
}

func showJsonFormatter(snapshot *bug.Snapshot) error {
	jsonBug := JSONBugSnapshot{
		snapshot.Id().String(),
		snapshot.Id().Human(),
		snapshot.CreatedAt,
		snapshot.LastEditTime(),
		snapshot.Status.String(),
		snapshot.Labels,
		snapshot.Title,
		JSONIdentity{},
		[]JSONIdentity{},
		[]JSONIdentity{},
		[]JSONComment{},
	}

	jsonBug.Author.Name = snapshot.Author.DisplayName()
	jsonBug.Author.Login = snapshot.Author.Login()
	jsonBug.Author.Id = snapshot.Author.Id().String()
	jsonBug.Author.HumanId = snapshot.Author.Id().Human()

	for _, element := range snapshot.Actors {
		jsonBug.Actors = append(jsonBug.Actors, JSONIdentity{
			element.Id().String(),
			element.Id().Human(),
			element.Name(),
			element.Login(),
		})
	}

	for _, element := range snapshot.Participants {
		jsonBug.Actors = append(jsonBug.Actors, JSONIdentity{
			element.Id().String(),
			element.Id().Human(),
			element.Name(),
			element.Login(),
		})
	}

	for i, comment := range snapshot.Comments {
		var message string
		if comment.Message == "" {
			message = "No description provided."
		} else {
			message = comment.Message
		}
		jsonBug.Comments = append(jsonBug.Comments, JSONComment{
			i,
			comment.Author.Name(),
			comment.Author.Login(),
			message,
		})
	}

	jsonObject, _ := json.MarshalIndent(jsonBug, "", "    ")
	fmt.Printf("%s\n", jsonObject)

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
	showCmd.Flags().StringVarP(&showFieldsQuery, "field", "", "",
		"Select field to display. Valid values are [author,authorEmail,createTime,lastEdit,humanId,id,labels,shortId,status,title,actors,participants]")
	showCmd.Flags().StringVarP(&showOutputFormat, "format", "f", "default",
		"Select the output formatting style. Valid values are [default,plain,json]")
}
