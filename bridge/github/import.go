package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/text"
)

const (
	keyOrigin      = "origin"
	keyGithubId    = "github-id"
	keyGithubUrl   = "github-url"
	keyGithubLogin = "github-login"
)

// githubImporter implement the Importer interface
type githubImporter struct {
	conf core.Configuration

	// iterator
	iterator *iterator

	// send only channel
	out chan<- core.ImportResult
}

func (gi *githubImporter) Init(conf core.Configuration) error {
	gi.conf = conf
	return nil
}

// ImportAll iterate over all the configured repository issues and ensure the creation of the
// missing issues / timeline items / edits / label events ...
func (gi *githubImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	gi.iterator = NewIterator(ctx, 10, gi.conf[keyOwner], gi.conf[keyProject], gi.conf[keyToken], since)
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(gi.out)

		// Loop over all matching issues
		for gi.iterator.NextIssue() {
			issue := gi.iterator.IssueValue()
			// create issue
			b, err := gi.ensureIssue(repo, issue)
			if err != nil {
				err := fmt.Errorf("issue creation: %v", err)
				out <- core.NewImportError(err, "")
				return
			}

			// loop over timeline items
			for gi.iterator.NextTimelineItem() {
				item := gi.iterator.TimelineItemValue()
				if err := gi.ensureTimelineItem(repo, b, item); err != nil {
					err := fmt.Errorf("timeline item creation: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			}

			// commit bug state
			if err := b.CommitAsNeeded(); err != nil {
				err = fmt.Errorf("bug commit: %v", err)
				out <- core.NewImportError(err, "")
				return
			}
		}

		if err := gi.iterator.Error(); err != nil && err != context.Canceled {
			gi.out <- core.NewImportError(err, "")
		}
	}()

	return out, nil
}

func (gi *githubImporter) ensureIssue(repo *cache.RepoCache, issue issueTimeline) (*cache.BugCache, error) {
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

	// get issue edits
	var issueEdits []userContentEdit
	for gi.iterator.NextIssueEdit() {
		issueEdits = append(issueEdits, gi.iterator.IssueEditValue())
	}

	// if issueEdits is empty
	if len(issueEdits) == 0 {
		if err == bug.ErrBugNotExist {
			cleanText, err := text.Cleanup(string(issue.Body))
			if err != nil {
				return nil, err
			}

			// create bug
			b, _, err = repo.NewBugRaw(
				author,
				issue.CreatedAt.Unix(),
				issue.Title,
				cleanText,
				nil,
				map[string]string{
					keyOrigin:    target,
					keyGithubId:  parseId(issue.Id),
					keyGithubUrl: issue.Url.String(),
				})
			if err != nil {
				return nil, err
			}

			// importing a new bug
			gi.out <- core.NewImportBug(b.Id())
		} else {
			gi.out <- core.NewImportNothing("", "bug already imported")
		}

	} else {
		// create bug from given issueEdits
		for i, edit := range issueEdits {
			if i == 0 && b != nil {
				// The first edit in the github result is the issue creation itself, we already have that
				gi.out <- core.NewImportNothing("", "bug already imported")
				continue
			}

			cleanText, err := text.Cleanup(string(*edit.Diff))
			if err != nil {
				return nil, err
			}

			// if the bug doesn't exist
			if b == nil {
				// we create the bug as soon as we have a legit first edition
				b, _, err = repo.NewBugRaw(
					author,
					issue.CreatedAt.Unix(),
					issue.Title,
					cleanText,
					nil,
					map[string]string{
						keyOrigin:    target,
						keyGithubId:  parseId(issue.Id),
						keyGithubUrl: issue.Url.String(),
					},
				)

				if err != nil {
					return nil, err
				}

				// importing a new bug
				gi.out <- core.NewImportBug(b.Id())
				continue
			}

			// other edits will be added as CommentEdit operations
			target, err := b.ResolveOperationWithMetadata(keyGithubId, parseId(issue.Id))
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

	switch item.Typename {
	case "IssueComment":
		// collect all comment edits
		var commentEdits []userContentEdit
		for gi.iterator.NextCommentEdit() {
			commentEdits = append(commentEdits, gi.iterator.CommentEditValue())
		}

		// ensureTimelineComment send import events over out chanel
		err := gi.ensureTimelineComment(repo, b, item.IssueComment, commentEdits)
		if err != nil {
			return fmt.Errorf("timeline comment creation: %v", err)
		}

	case "LabeledEvent":
		id := parseId(item.LabeledEvent.Id)
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err == nil {
			reason := fmt.Sprintf("operation already imported: %v", item.Typename)
			gi.out <- core.NewImportNothing("", reason)
			return nil
		}

		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(repo, item.LabeledEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.ForceChangeLabelsRaw(
			author,
			item.LabeledEvent.CreatedAt.Unix(),
			[]string{
				string(item.LabeledEvent.Label.Name),
			},
			nil,
			map[string]string{keyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportLabelChange(op.Id())
		return nil

	case "UnlabeledEvent":
		id := parseId(item.UnlabeledEvent.Id)
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err == nil {
			reason := fmt.Sprintf("operation already imported: %v", item.Typename)
			gi.out <- core.NewImportNothing("", reason)
			return nil
		}
		if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(repo, item.UnlabeledEvent.Actor)
		if err != nil {
			return err
		}

		op, err := b.ForceChangeLabelsRaw(
			author,
			item.UnlabeledEvent.CreatedAt.Unix(),
			nil,
			[]string{
				string(item.UnlabeledEvent.Label.Name),
			},
			map[string]string{keyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportLabelChange(op.Id())
		return nil

	case "ClosedEvent":
		id := parseId(item.ClosedEvent.Id)
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
			reason := fmt.Sprintf("operation already imported: %v", item.Typename)
			gi.out <- core.NewImportNothing("", reason)
			return nil
		}
		author, err := gi.ensurePerson(repo, item.ClosedEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.CloseRaw(
			author,
			item.ClosedEvent.CreatedAt.Unix(),
			map[string]string{keyGithubId: id},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(op.Id())
		return nil

	case "ReopenedEvent":
		id := parseId(item.ReopenedEvent.Id)
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
			reason := fmt.Sprintf("operation already imported: %v", item.Typename)
			gi.out <- core.NewImportNothing("", reason)
			return nil
		}
		author, err := gi.ensurePerson(repo, item.ReopenedEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.OpenRaw(
			author,
			item.ReopenedEvent.CreatedAt.Unix(),
			map[string]string{keyGithubId: id},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(op.Id())
		return nil

	case "RenamedTitleEvent":
		id := parseId(item.RenamedTitleEvent.Id)
		_, err := b.ResolveOperationWithMetadata(keyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
			reason := fmt.Sprintf("operation already imported: %v", item.Typename)
			gi.out <- core.NewImportNothing("", reason)
			return nil
		}
		author, err := gi.ensurePerson(repo, item.RenamedTitleEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.SetTitleRaw(
			author,
			item.RenamedTitleEvent.CreatedAt.Unix(),
			string(item.RenamedTitleEvent.CurrentTitle),
			map[string]string{keyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportTitleEdition(op.Id())
		return nil

	default:
		reason := fmt.Sprintf("ignoring timeline type: %v", item.Typename)
		gi.out <- core.NewImportNothing("", reason)
	}

	return nil
}

func (gi *githubImporter) ensureTimelineComment(repo *cache.RepoCache, b *cache.BugCache, item issueComment, edits []userContentEdit) error {
	// ensure person
	author, err := gi.ensurePerson(repo, item.Author)
	if err != nil {
		return err
	}

	targetOpID, err := b.ResolveOperationWithMetadata(keyGithubId, parseId(item.Id))
	if err == nil {
		gi.out <- core.NewImportNothing("", "comment already imported")
	} else if err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	// if no edits are given we create the comment
	if len(edits) == 0 {
		if err == cache.ErrNoMatchingOp {
			cleanText, err := text.Cleanup(string(item.Body))
			if err != nil {
				return err
			}

			// add comment operation
			op, err := b.AddCommentRaw(
				author,
				item.CreatedAt.Unix(),
				cleanText,
				nil,
				map[string]string{
					keyGithubId:  parseId(item.Id),
					keyGithubUrl: parseId(item.Url.String()),
				},
			)
			if err != nil {
				return err
			}

			gi.out <- core.NewImportComment(op.Id())
		}

	} else {
		for i, edit := range edits {
			if i == 0 && targetOpID != "" {
				// The first edit in the github result is the comment creation itself, we already have that
				gi.out <- core.NewImportNothing("", "comment already imported")
				continue
			}

			// ensure editor identity
			editor, err := gi.ensurePerson(repo, edit.Editor)
			if err != nil {
				return err
			}

			// create comment when target is empty
			if targetOpID == "" {
				cleanText, err := text.Cleanup(string(*edit.Diff))
				if err != nil {
					return err
				}

				op, err := b.AddCommentRaw(
					editor,
					edit.CreatedAt.Unix(),
					cleanText,
					nil,
					map[string]string{
						keyGithubId:  parseId(item.Id),
						keyGithubUrl: item.Url.String(),
					},
				)
				if err != nil {
					return err
				}

				// set target for the nexr edit now that the comment is created
				targetOpID = op.Id()
				continue
			}

			err = gi.ensureCommentEdit(repo, b, targetOpID, edit)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (gi *githubImporter) ensureCommentEdit(repo *cache.RepoCache, b *cache.BugCache, target entity.Id, edit userContentEdit) error {
	_, err := b.ResolveOperationWithMetadata(keyGithubId, parseId(edit.Id))
	if err == nil {
		gi.out <- core.NewImportNothing(b.Id(), "edition already imported")
		return nil
	}
	if err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	editor, err := gi.ensurePerson(repo, edit.Editor)
	if err != nil {
		return err
	}

	switch {
	case edit.DeletedAt != nil:
		// comment deletion, not supported yet
		gi.out <- core.NewImportNothing(b.Id(), "comment deletion is not supported yet")

	case edit.DeletedAt == nil:

		cleanText, err := text.Cleanup(string(*edit.Diff))
		if err != nil {
			return err
		}

		// comment edition
		op, err := b.EditCommentRaw(
			editor,
			edit.CreatedAt.Unix(),
			target,
			cleanText,
			map[string]string{
				keyGithubId: parseId(edit.Id),
			},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportCommentEdition(op.Id())
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
	if _, ok := err.(entity.ErrMultipleMatch); ok {
		return nil, err
	}

	// importing a new identity

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

	i, err = repo.NewIdentityRaw(
		name,
		email,
		string(actor.Login),
		string(actor.AvatarUrl),
		map[string]string{
			keyGithubLogin: string(actor.Login),
		},
	)

	if err != nil {
		return nil, err
	}

	gi.out <- core.NewImportIdentity(i.Id())
	return i, nil
}

func (gi *githubImporter) getGhost(repo *cache.RepoCache) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(keyGithubLogin, "ghost")
	if err == nil {
		return i, nil
	}
	if _, ok := err.(entity.ErrMultipleMatch); ok {
		return nil, err
	}

	var q ghostQuery

	variables := map[string]interface{}{
		"login": githubv4.String("ghost"),
	}

	gc := buildClient(gi.conf[keyToken])

	ctx, cancel := context.WithTimeout(gi.iterator.ctx, defaultTimeout)
	defer cancel()

	err = gc.Query(ctx, &q, variables)
	if err != nil {
		return nil, err
	}

	var name string
	if q.User.Name != nil {
		name = string(*q.User.Name)
	}

	return repo.NewIdentityRaw(
		name,
		"",
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
