package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/shurcooL/githubv4"
)

const keyGithubId = "github-id"
const keyGithubUrl = "github-url"

// githubImporter implement the Importer interface
type githubImporter struct {
	client *githubv4.Client
	conf   core.Configuration
	ghost  bug.Person
}

func (gi *githubImporter) Init(conf core.Configuration) error {
	gi.conf = conf
	gi.client = buildClient(conf)

	return gi.fetchGhost()
}

func (gi *githubImporter) ImportAll(repo *cache.RepoCache) error {
	q := &issueTimelineQuery{}
	variables := map[string]interface{}{
		"owner":         githubv4.String(gi.conf[keyUser]),
		"name":          githubv4.String(gi.conf[keyProject]),
		"issueFirst":    githubv4.Int(1),
		"issueAfter":    (*githubv4.String)(nil),
		"timelineFirst": githubv4.Int(10),
		"timelineAfter": (*githubv4.String)(nil),

		// Fun fact, github provide the comment edition in reverse chronological
		// order, because haha. Look at me, I'm dying of laughter.
		"issueEditLast":     githubv4.Int(10),
		"issueEditBefore":   (*githubv4.String)(nil),
		"commentEditLast":   githubv4.Int(10),
		"commentEditBefore": (*githubv4.String)(nil),
	}

	var b *cache.BugCache

	for {
		err := gi.client.Query(context.TODO(), &q, variables)
		if err != nil {
			return err
		}

		if len(q.Repository.Issues.Nodes) == 0 {
			return nil
		}

		issue := q.Repository.Issues.Nodes[0]

		if b == nil {
			b, err = gi.ensureIssue(repo, issue, variables)
			if err != nil {
				return err
			}
		}

		for _, itemEdge := range q.Repository.Issues.Nodes[0].Timeline.Edges {
			gi.ensureTimelineItem(b, itemEdge.Cursor, itemEdge.Node, variables)
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

func (gi *githubImporter) Import(repo *cache.RepoCache, id string) error {
	fmt.Println("IMPORT")

	return nil
}

func (gi *githubImporter) ensureIssue(repo *cache.RepoCache, issue issueTimeline, rootVariables map[string]interface{}) (*cache.BugCache, error) {
	fmt.Printf("import issue: %s\n", issue.Title)

	b, err := repo.ResolveBugCreateMetadata(keyGithubId, parseId(issue.Id))
	if err != nil && err != bug.ErrBugNotExist {
		return nil, err
	}

	// if there is no edit, the UserContentEdits given by github is empty. That
	// means that the original message is given by the issue message.
	//
	// if there is edits, the UserContentEdits given by github contains both the
	// original message and the following edits. The issue message give the last
	// version so we don't care about that.
	//
	// the tricky part: for an issue older than the UserContentEdits API, github
	// doesn't have the previous message version anymore and give an edition
	// with .Diff == nil. We have to filter them.

	if len(issue.UserContentEdits.Nodes) == 0 {
		if err == bug.ErrBugNotExist {
			b, err = repo.NewBugRaw(
				gi.makePerson(issue.Author),
				issue.CreatedAt.Unix(),
				// Todo: this might not be the initial title, we need to query the
				// timeline to be sure
				issue.Title,
				cleanupText(string(issue.Body)),
				nil,
				map[string]string{
					keyGithubId:  parseId(issue.Id),
					keyGithubUrl: issue.Url.String(),
				},
			)

			if err != nil {
				return nil, err
			}
		}

		return b, nil
	}

	// reverse the order, because github
	reverseEdits(issue.UserContentEdits.Nodes)

	for i, edit := range issue.UserContentEdits.Nodes {
		if b != nil && i == 0 {
			// The first edit in the github result is the creation itself, we already have that
			continue
		}

		if b == nil {
			if edit.Diff == nil {
				// not enough data given by github for old edit, ignore them
				continue
			}

			// we create the bug as soon as we have a legit first edition
			b, err = repo.NewBugRaw(
				gi.makePerson(issue.Author),
				issue.CreatedAt.Unix(),
				// Todo: this might not be the initial title, we need to query the
				// timeline to be sure
				issue.Title,
				cleanupText(string(*edit.Diff)),
				nil,
				map[string]string{
					keyGithubId:  parseId(issue.Id),
					keyGithubUrl: issue.Url.String(),
				},
			)
			if err != nil {
				return nil, err
			}
			continue
		}

		target, err := b.ResolveTargetWithMetadata(keyGithubId, parseId(issue.Id))
		if err != nil {
			return nil, err
		}

		err = gi.ensureCommentEdit(b, target, edit)
		if err != nil {
			return nil, err
		}
	}

	if !issue.UserContentEdits.PageInfo.HasNextPage {
		// if we still didn't get a legit edit, create the bug from the issue data
		if b == nil {
			return repo.NewBugRaw(
				gi.makePerson(issue.Author),
				issue.CreatedAt.Unix(),
				// Todo: this might not be the initial title, we need to query the
				// timeline to be sure
				issue.Title,
				cleanupText(string(issue.Body)),
				nil,
				map[string]string{
					keyGithubId:  parseId(issue.Id),
					keyGithubUrl: issue.Url.String(),
				},
			)
		}
		return b, nil
	}

	// We have more edit, querying them

	q := &issueEditQuery{}
	variables := map[string]interface{}{
		"owner":           rootVariables["owner"],
		"name":            rootVariables["name"],
		"issueFirst":      rootVariables["issueFirst"],
		"issueAfter":      rootVariables["issueAfter"],
		"issueEditLast":   githubv4.Int(10),
		"issueEditBefore": issue.UserContentEdits.PageInfo.StartCursor,
	}

	for {
		err := gi.client.Query(context.TODO(), &q, variables)
		if err != nil {
			return nil, err
		}

		edits := q.Repository.Issues.Nodes[0].UserContentEdits

		if len(edits.Nodes) == 0 {
			return b, nil
		}

		for _, edit := range edits.Nodes {
			if b == nil {
				if edit.Diff == nil {
					// not enough data given by github for old edit, ignore them
					continue
				}

				// we create the bug as soon as we have a legit first edition
				b, err = repo.NewBugRaw(
					gi.makePerson(issue.Author),
					issue.CreatedAt.Unix(),
					// Todo: this might not be the initial title, we need to query the
					// timeline to be sure
					issue.Title,
					cleanupText(string(*edit.Diff)),
					nil,
					map[string]string{
						keyGithubId:  parseId(issue.Id),
						keyGithubUrl: issue.Url.String(),
					},
				)
				if err != nil {
					return nil, err
				}
				continue
			}

			target, err := b.ResolveTargetWithMetadata(keyGithubId, parseId(issue.Id))
			if err != nil {
				return nil, err
			}

			err = gi.ensureCommentEdit(b, target, edit)
			if err != nil {
				return nil, err
			}
		}

		if !edits.PageInfo.HasNextPage {
			break
		}

		variables["issueEditBefore"] = edits.PageInfo.StartCursor
	}

	// TODO: check + import files

	// if we still didn't get a legit edit, create the bug from the issue data
	if b == nil {
		return repo.NewBugRaw(
			gi.makePerson(issue.Author),
			issue.CreatedAt.Unix(),
			// Todo: this might not be the initial title, we need to query the
			// timeline to be sure
			issue.Title,
			cleanupText(string(issue.Body)),
			nil,
			map[string]string{
				keyGithubId:  parseId(issue.Id),
				keyGithubUrl: issue.Url.String(),
			},
		)
	}

	return b, nil
}

func (gi *githubImporter) ensureTimelineItem(b *cache.BugCache, cursor githubv4.String, item timelineItem, rootVariables map[string]interface{}) error {
	fmt.Printf("import %s\n", item.Typename)

	switch item.Typename {
	case "IssueComment":
		return gi.ensureComment(b, cursor, item.IssueComment, rootVariables)

	case "LabeledEvent":
		id := parseId(item.LabeledEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		_, err = b.ChangeLabelsRaw(
			gi.makePerson(item.LabeledEvent.Actor),
			item.LabeledEvent.CreatedAt.Unix(),
			[]string{
				string(item.LabeledEvent.Label.Name),
			},
			nil,
			map[string]string{keyGithubId: id},
		)
		return err

	case "UnlabeledEvent":
		id := parseId(item.UnlabeledEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		_, err = b.ChangeLabelsRaw(
			gi.makePerson(item.UnlabeledEvent.Actor),
			item.UnlabeledEvent.CreatedAt.Unix(),
			nil,
			[]string{
				string(item.UnlabeledEvent.Label.Name),
			},
			map[string]string{keyGithubId: id},
		)
		return err

	case "ClosedEvent":
		id := parseId(item.ClosedEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		return b.CloseRaw(
			gi.makePerson(item.ClosedEvent.Actor),
			item.ClosedEvent.CreatedAt.Unix(),
			map[string]string{keyGithubId: id},
		)

	case "ReopenedEvent":
		id := parseId(item.ReopenedEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		return b.OpenRaw(
			gi.makePerson(item.ReopenedEvent.Actor),
			item.ReopenedEvent.CreatedAt.Unix(),
			map[string]string{keyGithubId: id},
		)

	case "RenamedTitleEvent":
		id := parseId(item.RenamedTitleEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		return b.SetTitleRaw(
			gi.makePerson(item.RenamedTitleEvent.Actor),
			item.RenamedTitleEvent.CreatedAt.Unix(),
			string(item.RenamedTitleEvent.CurrentTitle),
			map[string]string{keyGithubId: id},
		)

	default:
		fmt.Println("ignore event ", item.Typename)
	}

	return nil
}

func (gi *githubImporter) ensureComment(b *cache.BugCache, cursor githubv4.String, comment issueComment, rootVariables map[string]interface{}) error {
	target, err := b.ResolveTargetWithMetadata(keyGithubId, parseId(comment.Id))
	if err != nil && err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	// if there is no edit, the UserContentEdits given by github is empty. That
	// means that the original message is given by the comment message.
	//
	// if there is edits, the UserContentEdits given by github contains both the
	// original message and the following edits. The comment message give the last
	// version so we don't care about that.
	//
	// the tricky part: for a comment older than the UserContentEdits API, github
	// doesn't have the previous message version anymore and give an edition
	// with .Diff == nil. We have to filter them.

	if len(comment.UserContentEdits.Nodes) == 0 {
		if err == cache.ErrNoMatchingOp {
			err = b.AddCommentRaw(
				gi.makePerson(comment.Author),
				comment.CreatedAt.Unix(),
				cleanupText(string(comment.Body)),
				nil,
				map[string]string{
					keyGithubId: parseId(comment.Id),
				},
			)

			if err != nil {
				return err
			}
		}

		return nil
	}

	// reverse the order, because github
	reverseEdits(comment.UserContentEdits.Nodes)

	for i, edit := range comment.UserContentEdits.Nodes {
		if target != "" && i == 0 {
			// The first edit in the github result is the comment creation itself, we already have that
			continue
		}

		if target == "" {
			if edit.Diff == nil {
				// not enough data given by github for old edit, ignore them
				continue
			}

			err = b.AddCommentRaw(
				gi.makePerson(comment.Author),
				comment.CreatedAt.Unix(),
				cleanupText(string(*edit.Diff)),
				nil,
				map[string]string{
					keyGithubId:  parseId(comment.Id),
					keyGithubUrl: comment.Url.String(),
				},
			)
			if err != nil {
				return err
			}
		}

		err := gi.ensureCommentEdit(b, target, edit)
		if err != nil {
			return err
		}
	}

	if !comment.UserContentEdits.PageInfo.HasNextPage {
		return nil
	}

	// We have more edit, querying them

	q := &commentEditQuery{}
	variables := map[string]interface{}{
		"owner":             rootVariables["owner"],
		"name":              rootVariables["name"],
		"issueFirst":        rootVariables["issueFirst"],
		"issueAfter":        rootVariables["issueAfter"],
		"timelineFirst":     githubv4.Int(1),
		"timelineAfter":     cursor,
		"commentEditLast":   githubv4.Int(10),
		"commentEditBefore": comment.UserContentEdits.PageInfo.StartCursor,
	}

	for {
		err := gi.client.Query(context.TODO(), &q, variables)
		if err != nil {
			return err
		}

		edits := q.Repository.Issues.Nodes[0].Timeline.Nodes[0].IssueComment.UserContentEdits

		if len(edits.Nodes) == 0 {
			return nil
		}

		for i, edit := range edits.Nodes {
			if i == 0 {
				// The first edit in the github result is the creation itself, we already have that
				continue
			}

			err := gi.ensureCommentEdit(b, target, edit)
			if err != nil {
				return err
			}
		}

		if !edits.PageInfo.HasNextPage {
			break
		}

		variables["commentEditBefore"] = edits.PageInfo.StartCursor
	}

	// TODO: check + import files

	return nil
}

func (gi *githubImporter) ensureCommentEdit(b *cache.BugCache, target git.Hash, edit userContentEdit) error {
	if edit.Diff == nil {
		// this happen if the event is older than early 2018, Github doesn't have the data before that.
		// Best we can do is to ignore the event.
		return nil
	}

	if edit.Editor == nil {
		return fmt.Errorf("no editor")
	}

	_, err := b.ResolveTargetWithMetadata(keyGithubId, parseId(edit.Id))
	if err == nil {
		// already imported
		return nil
	}
	if err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	fmt.Println("import edition")

	switch {
	case edit.DeletedAt != nil:
		// comment deletion, not supported yet

	case edit.DeletedAt == nil:
		// comment edition
		err := b.EditCommentRaw(
			gi.makePerson(edit.Editor),
			edit.CreatedAt.Unix(),
			target,
			cleanupText(string(*edit.Diff)),
			map[string]string{
				keyGithubId: parseId(edit.Id),
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// makePerson create a bug.Person from the Github data
func (gi *githubImporter) makePerson(actor *actor) bug.Person {
	if actor == nil {
		return gi.ghost
	}

	return bug.Person{
		Name:      string(actor.Login),
		AvatarUrl: string(actor.AvatarUrl),
	}
}

func (gi *githubImporter) fetchGhost() error {
	var q userQuery

	variables := map[string]interface{}{
		"login": githubv4.String("ghost"),
	}

	err := gi.client.Query(context.TODO(), &q, variables)
	if err != nil {
		return err
	}

	gi.ghost = bug.Person{
		Name:      string(q.User.Login),
		AvatarUrl: string(q.User.AvatarUrl),
	}

	return nil
}

// parseId convert the unusable githubv4.ID (an interface{}) into a string
func parseId(id githubv4.ID) string {
	return fmt.Sprintf("%v", id)
}

func cleanupText(text string) string {
	// windows new line, Github, really ?
	text = strings.Replace(text, "\r\n", "\n", -1)

	// trim extra new line not displayed in the github UI but still present in the data
	return strings.TrimSpace(text)
}

func reverseEdits(edits []userContentEdit) []userContentEdit {
	for i, j := 0, len(edits)-1; i < j; i, j = i+1, j-1 {
		edits[i], edits[j] = edits[j], edits[i]
	}
	return edits
}
