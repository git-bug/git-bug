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
type githubImporter struct{}

func (*githubImporter) ImportAll(repo *cache.RepoCache, conf core.Configuration) error {
	client := buildClient(conf)

	q := &issueTimelineQuery{}
	variables := map[string]interface{}{
		"owner":         githubv4.String(conf[keyUser]),
		"name":          githubv4.String(conf[keyProject]),
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
		err := client.Query(context.TODO(), &q, variables)
		if err != nil {
			return err
		}

		if len(q.Repository.Issues.Nodes) == 0 {
			return nil
		}

		issue := q.Repository.Issues.Nodes[0]

		if b == nil {
			b, err = ensureIssue(repo, issue, client, variables)
			if err != nil {
				return err
			}
		}

		for _, itemEdge := range q.Repository.Issues.Nodes[0].Timeline.Edges {
			ensureTimelineItem(b, itemEdge.Cursor, itemEdge.Node, client, variables)
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

func ensureIssue(repo *cache.RepoCache, issue issueTimeline, client *githubv4.Client, rootVariables map[string]interface{}) (*cache.BugCache, error) {
	fmt.Printf("import issue: %s\n", issue.Title)

	b, err := repo.ResolveBugCreateMetadata(keyGithubId, parseId(issue.Id))
	if err != nil && err != bug.ErrBugNotExist {
		return nil, err
	}

	// if there is no edit, the UserContentEdits given by github is empty. That
	// means that the original message is given by the issue message.

	// if there is edits, the UserContentEdits given by github contains both the
	// original message and the following edits. The issue message give the last
	// version so we don't care about that.

	if len(issue.UserContentEdits.Nodes) == 0 {
		if err == bug.ErrBugNotExist {
			b, err = repo.NewBugRaw(
				makePerson(issue.Author),
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

	if err == bug.ErrBugNotExist {
		firstEdit := issue.UserContentEdits.Nodes[0]

		if firstEdit.Diff == nil {
			return nil, fmt.Errorf("no diff")
		}

		b, err = repo.NewBugRaw(
			makePerson(issue.Author),
			issue.CreatedAt.Unix(),
			// Todo: this might not be the initial title, we need to query the
			// timeline to be sure
			issue.Title,
			cleanupText(string(*firstEdit.Diff)),
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

	target, err := b.ResolveTargetWithMetadata(keyGithubId, parseId(issue.Id))
	if err != nil {
		return nil, err
	}

	for i, edit := range issue.UserContentEdits.Nodes {
		if i == 0 {
			// The first edit in the github result is the creation itself, we already have that
			continue
		}

		err := ensureCommentEdit(b, target, edit)
		if err != nil {
			return nil, err
		}
	}

	if !issue.UserContentEdits.PageInfo.HasNextPage {
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
		err := client.Query(context.TODO(), &q, variables)
		if err != nil {
			return nil, err
		}

		edits := q.Repository.Issues.Nodes[0].UserContentEdits

		if len(edits.Nodes) == 0 {
			return b, nil
		}

		for i, edit := range edits.Nodes {
			if i == 0 {
				// The first edit in the github result is the creation itself, we already have that
				continue
			}

			err := ensureCommentEdit(b, target, edit)
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

	return b, nil
}

func ensureTimelineItem(b *cache.BugCache, cursor githubv4.String, item timelineItem, client *githubv4.Client, rootVariables map[string]interface{}) error {
	fmt.Printf("import %s\n", item.Typename)

	switch item.Typename {
	case "IssueComment":
		return ensureComment(b, cursor, item.IssueComment, client, rootVariables)

	case "LabeledEvent":
		id := parseId(item.LabeledEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		_, err = b.ChangeLabelsRaw(
			makePerson(item.LabeledEvent.Actor),
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
			makePerson(item.UnlabeledEvent.Actor),
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
			makePerson(item.ClosedEvent.Actor),
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
			makePerson(item.ReopenedEvent.Actor),
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
			makePerson(item.RenamedTitleEvent.Actor),
			item.RenamedTitleEvent.CreatedAt.Unix(),
			string(item.RenamedTitleEvent.CurrentTitle),
			map[string]string{keyGithubId: id},
		)

	default:
		fmt.Println("ignore event ", item.Typename)
	}

	return nil
}

func ensureComment(b *cache.BugCache, cursor githubv4.String, comment issueComment, client *githubv4.Client, rootVariables map[string]interface{}) error {
	target, err := b.ResolveTargetWithMetadata(keyGithubId, parseId(comment.Id))
	if err != nil && err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	// if there is no edit, the UserContentEdits given by github is empty. That
	// means that the original message is given by the comment message.

	// if there is edits, the UserContentEdits given by github contains both the
	// original message and the following edits. The comment message give the last
	// version so we don't care about that.

	if len(comment.UserContentEdits.Nodes) == 0 {
		if err == cache.ErrNoMatchingOp {
			err = b.AddCommentRaw(
				makePerson(comment.Author),
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

	if err == cache.ErrNoMatchingOp {
		firstEdit := comment.UserContentEdits.Nodes[0]

		if firstEdit.Diff == nil {
			return fmt.Errorf("no diff")
		}

		err = b.AddCommentRaw(
			makePerson(comment.Author),
			comment.CreatedAt.Unix(),
			cleanupText(string(*firstEdit.Diff)),
			nil,
			map[string]string{
				keyGithubId:  parseId(comment.Id),
				keyGithubUrl: comment.Url.String(),
			},
		)
		if err != nil {
			return err
		}

		target, err = b.ResolveTargetWithMetadata(keyGithubId, parseId(comment.Id))
		if err != nil {
			return err
		}
	}

	for i, edit := range comment.UserContentEdits.Nodes {
		if i == 0 {
			// The first edit in the github result is the comment creation itself, we already have that
			continue
		}

		err := ensureCommentEdit(b, target, edit)
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
		err := client.Query(context.TODO(), &q, variables)
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

			err := ensureCommentEdit(b, target, edit)
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

func ensureCommentEdit(b *cache.BugCache, target git.Hash, edit userContentEdit) error {
	if edit.Editor == nil {
		return fmt.Errorf("no editor")
	}

	if edit.Diff == nil {
		return fmt.Errorf("no diff")
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

	fmt.Printf("import edition\n")

	switch {
	case edit.DeletedAt != nil:
		// comment deletion, not supported yet

	case edit.DeletedAt == nil:
		// comment edition
		err := b.EditCommentRaw(
			makePerson(*edit.Editor),
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
func makePerson(actor actor) bug.Person {
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
