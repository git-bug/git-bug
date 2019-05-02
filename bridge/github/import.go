package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/identity"
	"github.com/MichaelMure/git-bug/util/git"
	"github.com/shurcooL/githubv4"
)

const (
	keyGithubId    = "github-id"
	keyGithubUrl   = "github-url"
	keyGithubLogin = "github-login"
)

// githubImporter implement the Importer interface
type githubImporter struct {
	iterator *iterator
	conf     core.Configuration
}

func (gi *githubImporter) Init(conf core.Configuration) error {
	gi.conf = conf
	gi.iterator = newIterator(conf)
	return nil
}

// ImportAll .
func (gi *githubImporter) ImportAll(repo *cache.RepoCache, since time.Time) error {
	gi.iterator.since = since

	// Loop over all matching issues
	for gi.iterator.NextIssue() {
		issue := gi.iterator.IssueValue()

		fmt.Printf("importing issue: %v\n", gi.iterator.count)
		// get issue edits
		issueEdits := []userContentEdit{}
		for gi.iterator.NextIssueEdit() {
			if issueEdit := gi.iterator.IssueEditValue(); issueEdit.Diff != nil && string(*issueEdit.Diff) != "" {
				issueEdits = append(issueEdits, issueEdit)
			}
		}

		// create issue
		b, err := gi.ensureIssue(repo, issue, issueEdits)
		if err != nil {
			return fmt.Errorf("issue creation: %v", err)
		}

		// loop over timeline items
		for gi.iterator.NextTimeline() {
			item := gi.iterator.TimelineValue()

			// if item is comment
			if item.Typename == "IssueComment" {
				// collect all edits
				commentEdits := []userContentEdit{}
				for gi.iterator.NextCommentEdit() {
					if commentEdit := gi.iterator.CommentEditValue(); commentEdit.Diff != nil && string(*commentEdit.Diff) != "" {
						commentEdits = append(commentEdits, commentEdit)
					}
				}

				err := gi.ensureTimelineComment(repo, b, item.IssueComment, commentEdits)
				if err != nil {
					return fmt.Errorf("timeline event creation: %v", err)
				}

			} else {
				if err := gi.ensureTimelineItem(repo, b, item); err != nil {
					return fmt.Errorf("timeline comment creation: %v", err)
				}
			}
		}

		// commit bug state
		if err := b.CommitAsNeeded(); err != nil {
			return fmt.Errorf("bug commit: %v", err)
		}
	}

	if err := gi.iterator.Error(); err != nil {
		fmt.Printf("import error: %v\n", err)
		return err
	}

	fmt.Printf("Successfully imported %v issues from Github\n", gi.iterator.Count())
	return nil
}

func (gi *githubImporter) ensureIssue(repo *cache.RepoCache, issue issueTimeline, issueEdits []userContentEdit) (*cache.BugCache, error) {
	// ensure issue author
	author, err := gi.ensurePerson(repo, issue.Author)
	if err != nil {
		return nil, err
	}

	// resolve bug
	b, err := repo.ResolveBugCreateMetadata(keyGithubUrl, issue.Url.String())
	if err != nil && err != bug.ErrBugNotExist {
		return nil, err
	}

	// if issueEdits is empty
	if len(issueEdits) == 0 {
		if err == bug.ErrBugNotExist {
			// create bug
			b, err = repo.NewBugRaw(
				author,
				issue.CreatedAt.Unix(),
				issue.Title,
				cleanupText(string(issue.Body)),
				nil,
				map[string]string{
					keyGithubId:  parseId(issue.Id),
					keyGithubUrl: issue.Url.String(),
				})
			if err != nil {
				return nil, err
			}
		}

	} else {
		// create bug from given issueEdits
		for i, edit := range issueEdits {
			if i == 0 && b != nil {
				continue
			}

			// if the bug doesn't exist
			if b == nil {
				// we create the bug as soon as we have a legit first edition
				b, err = repo.NewBugRaw(
					author,
					issue.CreatedAt.Unix(),
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

			// other edits will be added as CommentEdit operations
			target, err := b.ResolveOperationWithMetadata(keyGithubUrl, issue.Url.String())
			if err != nil {
				return nil, err
			}

			err = gi.ensureCommentEdit(repo, b, target, edit)
			if err != nil {
				return nil, err
			}
		}
	}

	return b, nil
}

func (gi *githubImporter) ensureTimelineItem(repo *cache.RepoCache, b *cache.BugCache, item timelineItem) error {
	fmt.Printf("import event item: %s\n", item.Typename)

	switch item.Typename {
	case "IssueComment":

	case "LabeledEvent":
		id := parseId(item.LabeledEvent.Id)
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(repo, item.LabeledEvent.Actor)
		if err != nil {
			return err
		}
		_, _, err = b.ChangeLabelsRaw(
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
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(repo, item.UnlabeledEvent.Actor)
		if err != nil {
			return err
		}

		_, _, err = b.ChangeLabelsRaw(
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
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(repo, item.ClosedEvent.Actor)
		if err != nil {
			return err
		}
		_, err = b.CloseRaw(
			author,
			item.ClosedEvent.CreatedAt.Unix(),
			map[string]string{keyGithubId: id},
		)
		return err

	case "ReopenedEvent":
		id := parseId(item.ReopenedEvent.Id)
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(repo, item.ReopenedEvent.Actor)
		if err != nil {
			return err
		}
		_, err = b.OpenRaw(
			author,
			item.ReopenedEvent.CreatedAt.Unix(),
			map[string]string{keyGithubId: id},
		)
		return err

	case "RenamedTitleEvent":
		id := parseId(item.RenamedTitleEvent.Id)
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(repo, item.RenamedTitleEvent.Actor)
		if err != nil {
			return err
		}
		_, err = b.SetTitleRaw(
			author,
			item.RenamedTitleEvent.CreatedAt.Unix(),
			string(item.RenamedTitleEvent.CurrentTitle),
			map[string]string{keyGithubId: id},
		)
		return err

	default:
		fmt.Printf("ignore event: %v\n", item.Typename)
	}

	return nil
}

func (gi *githubImporter) ensureTimelineComment(repo *cache.RepoCache, b *cache.BugCache, item issueComment, edits []userContentEdit) error {
	// ensure person
	author, err := gi.ensurePerson(repo, item.Author)
	if err != nil {
		return err
	}

	var target git.Hash
	target, err = b.ResolveOperationWithMetadata(keyGithubId, parseId(item.Id))
	if err != nil && err != cache.ErrNoMatchingOp {
		// real error
		return err
	}
	// if no edits are given we create the comment
	if len(edits) == 0 {

		// if comment doesn't exist
		if err == cache.ErrNoMatchingOp {

			// add comment operation
			op, err := b.AddCommentRaw(
				author,
				item.CreatedAt.Unix(),
				cleanupText(string(item.Body)),
				nil,
				map[string]string{
					keyGithubId:  parseId(item.Id),
					keyGithubUrl: parseId(item.Url.String()),
				},
			)
			if err != nil {
				return err
			}

			// set hash
			target, err = op.Hash()
			if err != nil {
				return err
			}
		}
	} else {
		for i, edit := range item.UserContentEdits.Nodes {
			if i == 0 && target != "" {
				continue
			}

			// ensure editor identity
			editor, err := gi.ensurePerson(repo, edit.Editor)
			if err != nil {
				return err
			}

			// create comment when target is empty
			if target == "" {
				op, err := b.AddCommentRaw(
					editor,
					edit.CreatedAt.Unix(),
					cleanupText(string(*edit.Diff)),
					nil,
					map[string]string{
						keyGithubId:  parseId(item.Id),
						keyGithubUrl: item.Url.String(),
					},
				)
				if err != nil {
					return err
				}

				// set hash
				target, err = op.Hash()
				if err != nil {
					return err
				}

				continue
			}

			err = gi.ensureCommentEdit(repo, b, target, edit)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (gi *githubImporter) ensureCommentEdit(repo *cache.RepoCache, b *cache.BugCache, target git.Hash, edit userContentEdit) error {
	_, err := b.ResolveOperationWithMetadata(keyGithubId, parseId(edit.Id))
	if err == nil {
		// already imported
		return nil
	}
	if err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	fmt.Println("import edition")

	editor, err := gi.ensurePerson(repo, edit.Editor)
	if err != nil {
		return err
	}

	switch {
	case edit.DeletedAt != nil:
		// comment deletion, not supported yet
		fmt.Println("comment deletion ....")

	case edit.DeletedAt == nil:

		// comment edition
		_, err := b.EditCommentRaw(
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

// ensurePerson create a bug.Person from the Github data
func (gi *githubImporter) ensurePerson(repo *cache.RepoCache, actor *actor) (*cache.IdentityCache, error) {
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

	err = gi.iterator.gc.Query(context.TODO(), &q, variables)
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
