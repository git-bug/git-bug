package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/shurcooL/githubv4"
)

const keyGithubId = "github-id"
const keyGithubUrl = "github-url"
const keyGithubLogin = "github-login"

// githubImporter implement the Importer interface
type githubImporter struct {
	client *githubv4.Client
	conf   core.Configuration
}

func (gi *githubImporter) Init(conf core.Configuration) error {
	gi.conf = conf
	gi.client = buildClient(conf)

	return nil
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
			err = gi.ensureTimelineItem(repo, b, itemEdge.Cursor, itemEdge.Node, variables)
			if err != nil {
				return err
			}
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

	author, err := gi.makePerson(repo, issue.Author)
	if err != nil {
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
				author,
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
				author,
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

		err = gi.ensureCommentEdit(repo, b, target, edit)
		if err != nil {
			return nil, err
		}
	}

	if !issue.UserContentEdits.PageInfo.HasNextPage {
		// if we still didn't get a legit edit, create the bug from the issue data
		if b == nil {
			return repo.NewBugRaw(
				author,
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
					author,
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

			err = gi.ensureCommentEdit(repo, b, target, edit)
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
			author,
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

func (gi *githubImporter) ensureTimelineItem(repo *cache.RepoCache, b *cache.BugCache, cursor githubv4.String, item timelineItem, rootVariables map[string]interface{}) error {
	fmt.Printf("import %s\n", item.Typename)

	switch item.Typename {
	case "IssueComment":
		return gi.ensureComment(repo, b, cursor, item.IssueComment, rootVariables)

	case "LabeledEvent":
		id := parseId(item.LabeledEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.makePerson(repo, item.LabeledEvent.Actor)
		if err != nil {
			return err
		}
		_, err = b.ChangeLabelsRaw(
			author,
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
		author, err := gi.makePerson(repo, item.UnlabeledEvent.Actor)
		if err != nil {
			return err
		}
		_, err = b.ChangeLabelsRaw(
			author,
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
		author, err := gi.makePerson(repo, item.ClosedEvent.Actor)
		if err != nil {
			return err
		}
		return b.CloseRaw(
			author,
			item.ClosedEvent.CreatedAt.Unix(),
			map[string]string{keyGithubId: id},
		)

	case "ReopenedEvent":
		id := parseId(item.ReopenedEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.makePerson(repo, item.ReopenedEvent.Actor)
		if err != nil {
			return err
		}
		return b.OpenRaw(
			author,
			item.ReopenedEvent.CreatedAt.Unix(),
			map[string]string{keyGithubId: id},
		)

	case "RenamedTitleEvent":
		id := parseId(item.RenamedTitleEvent.Id)
		_, err := b.ResolveTargetWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.makePerson(repo, item.RenamedTitleEvent.Actor)
		if err != nil {
			return err
		}
		return b.SetTitleRaw(
			author,
			item.RenamedTitleEvent.CreatedAt.Unix(),
			string(item.RenamedTitleEvent.CurrentTitle),
			map[string]string{keyGithubId: id},
		)

	default:
		fmt.Println("ignore event ", item.Typename)
	}

	return nil
}

func (gi *githubImporter) ensureComment(repo *cache.RepoCache, b *cache.BugCache, cursor githubv4.String, comment issueComment, rootVariables map[string]interface{}) error {
	target, err := b.ResolveTargetWithMetadata(keyGithubId, parseId(comment.Id))
	if err != nil && err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	author, err := gi.makePerson(repo, comment.Author)
	if err != nil {
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
				author,
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
				author,
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

		err := gi.ensureCommentEdit(repo, b, target, edit)
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

			err := gi.ensureCommentEdit(repo, b, target, edit)
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

func (gi *githubImporter) ensureCommentEdit(repo *cache.RepoCache, b *cache.BugCache, target git.Hash, edit userContentEdit) error {
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

	editor, err := gi.makePerson(repo, edit.Editor)
	if err != nil {
		return err
	}

	switch {
	case edit.DeletedAt != nil:
		// comment deletion, not supported yet

	case edit.DeletedAt == nil:
		// comment edition
		err := b.EditCommentRaw(
			editor,
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
func (gi *githubImporter) makePerson(repo *cache.RepoCache, actor *actor) (*cache.IdentityCache, error) {
	// When a user has been deleted, Github return a null actor, while displaying a profile named "ghost"
	// in it's UI. So we need a special case to get it.
	if actor == nil {
		return gi.getGhost(repo)
	}

	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(keyGithubLogin, string(actor.Login))
	if err == nil {
		return i, nil
	}
	if _, ok := err.(identity.ErrMultipleMatch); ok {
		return nil, err
	}

	var name string
	var email string

	switch actor.Typename {
	case "User":
		if actor.User.Name != nil {
			name = string(*(actor.User.Name))
		}
		email = string(actor.User.Email)
	case "Organization":
		if actor.Organization.Name != nil {
			name = string(*(actor.Organization.Name))
		}
		if actor.Organization.Email != nil {
			email = string(*(actor.Organization.Email))
		}
	case "Bot":
	}

	return repo.NewIdentityRaw(
		name,
		email,
		string(actor.Login),
		string(actor.AvatarUrl),
		map[string]string{
			keyGithubLogin: string(actor.Login),
		},
	)
}

func (gi *githubImporter) getGhost(repo *cache.RepoCache) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(keyGithubLogin, "ghost")
	if err == nil {
		return i, nil
	}
	if _, ok := err.(identity.ErrMultipleMatch); ok {
		return nil, err
	}

	var q userQuery

	variables := map[string]interface{}{
		"login": githubv4.String("ghost"),
	}

	err = gi.client.Query(context.TODO(), &q, variables)
	if err != nil {
		return nil, err
	}

	var name string
	if q.User.Name != nil {
		name = string(*q.User.Name)
	}

	return repo.NewIdentityRaw(
		name,
		string(q.User.Email),
		string(q.User.Login),
		string(q.User.AvatarUrl),
		map[string]string{
			keyGithubLogin: string(q.User.Login),
		},
	)
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
