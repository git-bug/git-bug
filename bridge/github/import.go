package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bridge/core/auth"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
	"github.com/MichaelMure/git-bug/util/text"
)

// githubImporter implement the Importer interface
type githubImporter struct {
	conf core.Configuration

	// default client
	client *githubv4.Client

	// iterator
	iterator *iterator

	// send only channel
	out chan<- core.ImportResult
}

func (gi *githubImporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	gi.conf = conf

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrMissingIdentityToken
	}

	gi.client = buildClient(creds[0].(*auth.Token))

	return nil
}

// ImportAll iterate over all the configured repository issues and ensure the creation of the
// missing issues / timeline items / edits / label events ...
func (gi *githubImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	gi.iterator = NewIterator(ctx, gi.client, 10, gi.conf[confKeyOwner], gi.conf[confKeyProject], since)
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
				err := gi.ensureTimelineItem(repo, b, item)
				if err != nil {
					err = fmt.Errorf("timeline item creation: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			}

			if !b.NeedCommit() {
				out <- core.NewImportNothing(b.Id(), "no imported operation")
			} else if err := b.Commit(); err != nil {
				// commit bug state
				err = fmt.Errorf("bug commit: %v", err)
				out <- core.NewImportError(err, "")
				return
			}
		}

		if err := gi.iterator.Error(); err != nil {
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
	b, err := repo.ResolveBugMatcher(func(excerpt *cache.BugExcerpt) bool {
		return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
			excerpt.CreateMetadata[metaKeyGithubId] == parseId(issue.Id)
	})
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
					core.MetaKeyOrigin: target,
					metaKeyGithubId:    parseId(issue.Id),
					metaKeyGithubUrl:   issue.Url.String(),
				})
			if err != nil {
				return nil, err
			}

			// importing a new bug
			gi.out <- core.NewImportBug(b.Id())
		}
	} else {
		// create bug from given issueEdits
		for i, edit := range issueEdits {
			if i == 0 && b != nil {
				// The first edit in the github result is the issue creation itself, we already have that
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
					issue.Title, // TODO: this is the *current* title, not the original one
					cleanText,
					nil,
					map[string]string{
						core.MetaKeyOrigin: target,
						metaKeyGithubId:    parseId(issue.Id),
						metaKeyGithubUrl:   issue.Url.String(),
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
			target, err := b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(issue.Id))
			if err == cache.ErrNoMatchingOp {
				// original comment is missing somehow, issuing a warning
				gi.out <- core.NewImportWarning(fmt.Errorf("comment ID %s to edit is missing", parseId(issue.Id)), b.Id())
				continue
			}
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
		return nil

	case "LabeledEvent":
		id := parseId(item.LabeledEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err == nil {
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
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportLabelChange(op.Id())
		return nil

	case "UnlabeledEvent":
		id := parseId(item.UnlabeledEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err == nil {
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
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportLabelChange(op.Id())
		return nil

	case "ClosedEvent":
		id := parseId(item.ClosedEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
			return nil
		}
		author, err := gi.ensurePerson(repo, item.ClosedEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.CloseRaw(
			author,
			item.ClosedEvent.CreatedAt.Unix(),
			map[string]string{metaKeyGithubId: id},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(op.Id())
		return nil

	case "ReopenedEvent":
		id := parseId(item.ReopenedEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
			return nil
		}
		author, err := gi.ensurePerson(repo, item.ReopenedEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.OpenRaw(
			author,
			item.ReopenedEvent.CreatedAt.Unix(),
			map[string]string{metaKeyGithubId: id},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(op.Id())
		return nil

	case "RenamedTitleEvent":
		id := parseId(item.RenamedTitleEvent.Id)
		_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
		if err != cache.ErrNoMatchingOp {
			return err
		}
		if err == nil {
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
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportTitleEdition(op.Id())
		return nil
	}

	return nil
}

func (gi *githubImporter) ensureTimelineComment(repo *cache.RepoCache, b *cache.BugCache, item issueComment, edits []userContentEdit) error {
	// ensure person
	author, err := gi.ensurePerson(repo, item.Author)
	if err != nil {
		return err
	}

	targetOpID, err := b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(item.Id))
	if err != nil && err != cache.ErrNoMatchingOp {
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
					metaKeyGithubId:  parseId(item.Id),
					metaKeyGithubUrl: parseId(item.Url.String()),
				},
			)
			if err != nil {
				return err
			}

			gi.out <- core.NewImportComment(op.Id())
			return nil
		}

	} else {
		for i, edit := range edits {
			if i == 0 && targetOpID != "" {
				// The first edit in the github result is the comment creation itself, we already have that
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
						metaKeyGithubId:  parseId(item.Id),
						metaKeyGithubUrl: item.Url.String(),
					},
				)
				if err != nil {
					return err
				}
				gi.out <- core.NewImportComment(op.Id())

				// set target for the next edit now that the comment is created
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
	_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(edit.Id))
	if err == nil {
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
		return nil

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
				metaKeyGithubId: parseId(edit.Id),
			},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportCommentEdition(op.Id())
		return nil
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
	i, err := repo.ResolveIdentityImmutableMetadata(metaKeyGithubLogin, string(actor.Login))
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
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

	// Name is not necessarily set, fallback to login as a name is required in the identity
	if name == "" {
		name = string(actor.Login)
	}

	i, err = repo.NewIdentityRaw(
		name,
		email,
		string(actor.Login),
		string(actor.AvatarUrl),
		nil,
		map[string]string{
			metaKeyGithubLogin: string(actor.Login),
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
	i, err := repo.ResolveIdentityImmutableMetadata(metaKeyGithubLogin, "ghost")
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}

	var q ghostQuery

	variables := map[string]interface{}{
		"login": githubv4.String("ghost"),
	}

	ctx, cancel := context.WithTimeout(gi.iterator.ctx, defaultTimeout)
	defer cancel()

	err = gi.client.Query(ctx, &q, variables)
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
		nil,
		map[string]string{
			metaKeyGithubLogin: string(q.User.Login),
		},
	)
}

// parseId convert the unusable githubv4.ID (an interface{}) into a string
func parseId(id githubv4.ID) string {
	return fmt.Sprintf("%v", id)
}
