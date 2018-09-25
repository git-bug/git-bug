package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/shurcooL/githubv4"
)

const keyGithubId = "github-id"
const keyGithubUrl = "github-url"

// githubImporter implement the Importer interface
type githubImporter struct{}

type Actor struct {
	Login     githubv4.String
	AvatarUrl githubv4.String
}

type ActorEvent struct {
	Id        githubv4.ID
	CreatedAt githubv4.DateTime
	Actor     Actor
}

type AuthorEvent struct {
	Id        githubv4.ID
	CreatedAt githubv4.DateTime
	Author    Actor
}

type TimelineItem struct {
	Typename githubv4.String `graphql:"__typename"`

	// Issue
	IssueComment struct {
		AuthorEvent
		Body githubv4.String
		Url  githubv4.URI
		// TODO: edition
	} `graphql:"... on IssueComment"`

	// Label
	LabeledEvent struct {
		ActorEvent
		Label struct {
			// Color githubv4.String
			Name githubv4.String
		}
	} `graphql:"... on LabeledEvent"`
	UnlabeledEvent struct {
		ActorEvent
		Label struct {
			// Color githubv4.String
			Name githubv4.String
		}
	} `graphql:"... on UnlabeledEvent"`

	// Status
	ClosedEvent struct {
		ActorEvent
		// Url githubv4.URI
	} `graphql:"... on  ClosedEvent"`
	ReopenedEvent struct {
		ActorEvent
	} `graphql:"... on  ReopenedEvent"`

	// Title
	RenamedTitleEvent struct {
		ActorEvent
		CurrentTitle  githubv4.String
		PreviousTitle githubv4.String
	} `graphql:"... on RenamedTitleEvent"`
}

type Issue struct {
	AuthorEvent
	Title string
	Body  githubv4.String
	Url   githubv4.URI

	Timeline struct {
		Nodes    []TimelineItem
		PageInfo struct {
			EndCursor   githubv4.String
			HasNextPage bool
		}
	} `graphql:"timeline(first: $timelineFirst, after: $timelineAfter)"`
}

var q struct {
	Repository struct {
		Issues struct {
			Nodes    []Issue
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
		} `graphql:"issues(first: $issueFirst, after: $issueAfter, orderBy: {field: CREATED_AT, direction: ASC})"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

func (*githubImporter) ImportAll(repo *cache.RepoCache, conf core.Configuration) error {
	client := buildClient(conf)

	variables := map[string]interface{}{
		"owner":         githubv4.String(conf[keyUser]),
		"name":          githubv4.String(conf[keyProject]),
		"issueFirst":    githubv4.Int(1),
		"issueAfter":    (*githubv4.String)(nil),
		"timelineFirst": githubv4.Int(10),
		"timelineAfter": (*githubv4.String)(nil),
	}

	var b *cache.BugCache

	for {
		err := client.Query(context.TODO(), &q, variables)
		if err != nil {
			return err
		}

		if len(q.Repository.Issues.Nodes) != 1 {
			return fmt.Errorf("Something went wrong when iterating issues, len is %d", len(q.Repository.Issues.Nodes))
		}

		issue := q.Repository.Issues.Nodes[0]

		if b == nil {
			b, err = importIssue(repo, issue)
			if err != nil {
				return err
			}
		}

		for _, item := range q.Repository.Issues.Nodes[0].Timeline.Nodes {
			importTimelineItem(b, item)
		}

		if !issue.Timeline.PageInfo.HasNextPage {
			err = b.CommitAsNeeded()
			if err != nil {
				return err
			}

			b = nil

			if !q.Repository.Issues.PageInfo.HasNextPage {
				break
			}

			variables["issueAfter"] = githubv4.NewString(q.Repository.Issues.PageInfo.EndCursor)
			variables["timelineAfter"] = (*githubv4.String)(nil)
			continue
		}

		variables["timelineAfter"] = githubv4.NewString(issue.Timeline.PageInfo.EndCursor)
	}

	return nil
}

func (*githubImporter) Import(repo *cache.RepoCache, conf core.Configuration, id string) error {
	fmt.Println(conf)
	fmt.Println("IMPORT")

	return nil
}

func importIssue(repo *cache.RepoCache, issue Issue) (*cache.BugCache, error) {
	fmt.Printf("import issue: %s\n", issue.Title)

	// TODO: check + import files

	return repo.NewBugRaw(
		makePerson(issue.Author),
		issue.CreatedAt.Unix(),
		issue.Title,
		cleanupText(string(issue.Body)),
		nil,
		map[string]string{
			keyGithubId:  parseId(issue.Id),
			keyGithubUrl: issue.Url.String(),
		},
	)
}

func importTimelineItem(b *cache.BugCache, item TimelineItem) error {
	switch item.Typename {
	case "IssueComment":
		// fmt.Printf("import %s: %s\n", item.Typename, item.IssueComment)
		return b.AddCommentRaw(
			makePerson(item.IssueComment.Author),
			item.IssueComment.CreatedAt.Unix(),
			cleanupText(string(item.IssueComment.Body)),
			nil,
			map[string]string{
				keyGithubId:  parseId(item.IssueComment.Id),
				keyGithubUrl: item.IssueComment.Url.String(),
			},
		)

	case "LabeledEvent":
		// fmt.Printf("import %s: %s\n", item.Typename, item.LabeledEvent)
		_, err := b.ChangeLabelsRaw(
			makePerson(item.LabeledEvent.Actor),
			item.LabeledEvent.CreatedAt.Unix(),
			[]string{
				string(item.LabeledEvent.Label.Name),
			},
			nil,
		)
		return err

	case "UnlabeledEvent":
		// fmt.Printf("import %s: %s\n", item.Typename, item.UnlabeledEvent)
		_, err := b.ChangeLabelsRaw(
			makePerson(item.UnlabeledEvent.Actor),
			item.UnlabeledEvent.CreatedAt.Unix(),
			nil,
			[]string{
				string(item.UnlabeledEvent.Label.Name),
			},
		)
		return err

	case "ClosedEvent":
		// fmt.Printf("import %s: %s\n", item.Typename, item.ClosedEvent)
		return b.CloseRaw(
			makePerson(item.ClosedEvent.Actor),
			item.ClosedEvent.CreatedAt.Unix(),
		)

	case "ReopenedEvent":
		// fmt.Printf("import %s: %s\n", item.Typename, item.ReopenedEvent)
		return b.OpenRaw(
			makePerson(item.ReopenedEvent.Actor),
			item.ReopenedEvent.CreatedAt.Unix(),
		)

	case "RenamedTitleEvent":
		// fmt.Printf("import %s: %s\n", item.Typename, item.RenamedTitleEvent)
		return b.SetTitleRaw(
			makePerson(item.RenamedTitleEvent.Actor),
			item.RenamedTitleEvent.CreatedAt.Unix(),
			string(item.RenamedTitleEvent.CurrentTitle),
		)

	default:
		fmt.Println("ignore event ", item.Typename)
	}

	return nil
}

// makePerson create a bug.Person from the Github data
func makePerson(actor Actor) bug.Person {
	return bug.Person{
		Name:      string(actor.Login),
		AvatarUrl: string(actor.AvatarUrl),
	}
}

// parseId convert the unusable githubv4.ID (an interface{}) into a string
func parseId(id githubv4.ID) string {
	return fmt.Sprintf("%v", id)
}

func cleanupText(text string) string {
	// windows new line, Github, really ?
	return strings.Replace(text, "\r\n", "\n", -1)
}
