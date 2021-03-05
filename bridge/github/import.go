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

const EMPTY_TITLE_PLACEHOLDER = "<empty string>"

// githubImporter implement the Importer interface
type githubImporter struct {
	conf core.Configuration

	// mediator to access the Github API
	mediator *importMediator

	// send only channel
	out chan<- core.ImportResult
}

func (gi *githubImporter) Init(_ context.Context, _ *cache.RepoCache, conf core.Configuration) error {
	gi.conf = conf
	return nil
}

// ImportAll iterate over all the configured repository issues and ensure the creation of the
// missing issues / timeline items / edits / label events ...
func (gi *githubImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyLogin, gi.conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return nil, err
	}
	if len(creds) <= 0 {
		return nil, ErrMissingIdentityToken
	}
	client := buildClient(creds[0].(*auth.Token))
	gi.mediator = NewImportMediator(ctx, client, gi.conf[confKeyOwner], gi.conf[confKeyProject], since)
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(gi.out)

		// Loop over all matching issues
		for issue := range gi.mediator.Issues() {
			// create issue
			b, err := gi.ensureIssue(ctx, repo, &issue)
			if err != nil {
				err := fmt.Errorf("issue creation: %v", err)
				out <- core.NewImportError(err, "")
				return
			}

			// loop over timeline items
			for item := range gi.mediator.TimelineItems(&issue) {
				err := gi.ensureTimelineItem(ctx, repo, b, item)
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

		if err := gi.mediator.Error(); err != nil {
			gi.out <- core.NewImportError(err, "")
		}
	}()

	return out, nil
}

func (gi *githubImporter) ensureIssue(ctx context.Context, repo *cache.RepoCache, issue *issue) (*cache.BugCache, error) {
	author, err := gi.ensurePerson(ctx, repo, issue.Author)
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

	// get first issue edit
	// if it exists, then it holds the bug creation
	firstEdit, hasEdit := <-gi.mediator.IssueEdits(issue)

	// At Github there exist issues with seemingly empty titles. An example is
	// https://github.com/NixOS/nixpkgs/issues/72730 .
	// The title provided by the GraphQL API actually consists of a space followed by a
	// zero width space (U+200B). This title would cause the NewBugRaw() function to
	// return an error: empty title.
	title := string(issue.Title)
	if title == " \u200b" { // U+200B == zero width space
		title = EMPTY_TITLE_PLACEHOLDER
	}

	if err == bug.ErrBugNotExist {
		var textInput string
		if hasEdit {
			// use the first issue edit: it represents the bug creation itself
			textInput = string(*firstEdit.Diff)
		} else {
			// if there are no issue edits then the issue struct holds the bug creation
			textInput = string(issue.Body)
		}
		cleanText, err := text.Cleanup(textInput)
		if err != nil {
			return nil, err
		}
		// create bug
		b, _, err = repo.NewBugRaw(
			author,
			issue.CreatedAt.Unix(),
			title, // TODO: this is the *current* title, not the original one
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
	if b == nil {
		return nil, fmt.Errorf("finding or creating issue")
	}
	// process remaining issue edits, if they exist
	for edit := range gi.mediator.IssueEdits(issue) {
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

		err = gi.ensureCommentEdit(ctx, repo, b, target, edit)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (gi *githubImporter) ensureTimelineItem(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, item timelineItem) error {

	switch item.Typename {
	case "IssueComment":
		err := gi.ensureComment(ctx, repo, b, &item.IssueComment)
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
		author, err := gi.ensurePerson(ctx, repo, item.LabeledEvent.Actor)
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
		author, err := gi.ensurePerson(ctx, repo, item.UnlabeledEvent.Actor)
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
		author, err := gi.ensurePerson(ctx, repo, item.ClosedEvent.Actor)
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
		author, err := gi.ensurePerson(ctx, repo, item.ReopenedEvent.Actor)
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
		author, err := gi.ensurePerson(ctx, repo, item.RenamedTitleEvent.Actor)
		if err != nil {
			return err
		}

		// At Github there exist issues with seemingly empty titles. An example is
		// https://github.com/NixOS/nixpkgs/issues/72730 .
		// The title provided by the GraphQL API actually consists of a space followed
		// by a zero width space (U+200B). This title would cause the NewBugRaw()
		// function to return an error: empty title.
		title := string(item.RenamedTitleEvent.CurrentTitle)
		if title == " \u200b" { // U+200B == zero width space
			title = EMPTY_TITLE_PLACEHOLDER
		}

		op, err := b.SetTitleRaw(
			author,
			item.RenamedTitleEvent.CreatedAt.Unix(),
			title,
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

func (gi *githubImporter) ensureComment(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, comment *issueComment) error {
	author, err := gi.ensurePerson(ctx, repo, comment.Author)
	if err != nil {
		return err
	}

	targetOpID, err := b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(comment.Id))
	if err != nil && err != cache.ErrNoMatchingOp {
		// real error
		return err
	}
	firstEdit, hasEdit := <-gi.mediator.CommentEdits(comment)
	if err == cache.ErrNoMatchingOp {
		var textInput string
		if hasEdit {
			// use the first comment edit: it represents the comment creation itself
			textInput = string(*firstEdit.Diff)
		} else {
			// if there are not comment edits, then the comment struct holds the comment creation
			textInput = string(comment.Body)
		}
		cleanText, err := text.Cleanup(textInput)
		if err != nil {
			return err
		}

		// add comment operation
		op, err := b.AddCommentRaw(
			author,
			comment.CreatedAt.Unix(),
			cleanText,
			nil,
			map[string]string{
				metaKeyGithubId:  parseId(comment.Id),
				metaKeyGithubUrl: comment.Url.String(),
			},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportComment(op.Id())
		targetOpID = op.Id()
	}
	if targetOpID == "" {
		return fmt.Errorf("finding or creating issue comment")
	}
	// process remaining comment edits, if they exist
	for edit := range gi.mediator.CommentEdits(comment) {
		// ensure editor identity
		_, err := gi.ensurePerson(ctx, repo, edit.Editor)
		if err != nil {
			return err
		}

		err = gi.ensureCommentEdit(ctx, repo, b, targetOpID, edit)
		if err != nil {
			return err
		}
	}
	return nil
}

func (gi *githubImporter) ensureCommentEdit(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, target entity.Id, edit userContentEdit) error {
	_, err := b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(edit.Id))
	if err == nil {
		return nil
	}
	if err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	editor, err := gi.ensurePerson(ctx, repo, edit.Editor)
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
func (gi *githubImporter) ensurePerson(ctx context.Context, repo *cache.RepoCache, actor *actor) (*cache.IdentityCache, error) {
	// When a user has been deleted, Github return a null actor, while displaying a profile named "ghost"
	// in it's UI. So we need a special case to get it.
	if actor == nil {
		return gi.getGhost(ctx, repo)
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

func (gi *githubImporter) getGhost(ctx context.Context, repo *cache.RepoCache) (*cache.IdentityCache, error) {
	loginName := "ghost"
	// Look first in the cache
	i, err := repo.ResolveIdentityImmutableMetadata(metaKeyGithubLogin, loginName)
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}
	user, err := gi.mediator.User(ctx, loginName)
	userName := ""
	if user.Name != nil {
		userName = string(*user.Name)
	}
	return repo.NewIdentityRaw(
		userName,
		"",
		string(user.Login),
		string(user.AvatarUrl),
		map[string]string{
			metaKeyGithubLogin: string(user.Login),
		},
	)
}

// parseId converts the unusable githubv4.ID (an interface{}) into a string
func parseId(id githubv4.ID) string {
	return fmt.Sprintf("%v", id)
}
