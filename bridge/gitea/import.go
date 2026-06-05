package gitea

import (
	"context"
	"fmt"
	"strconv"
	"time"

	gitea "gitea.dev/sdk"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/gitea/iterator"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/repository"
	"github.com/git-bug/git-bug/util/text"
)

// implements the Importer interface
type giteaImporter struct {
	conf core.Configuration

	// default client
	client *gitea.Client

	// iterator
	iterator *iterator.Iterator

	// send only channel
	out chan<- core.ImportResult
}

const DeletedIdentity = "@deleted-user"

func (gi *giteaImporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	gi.conf = conf

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyBaseURL]),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrMissingIdentityToken
	}

	gi.client, err = buildClient(conf[confKeyBaseURL], creds[0].(*auth.Token))
	if err != nil {
		return err
	}

	return nil
}

// ImportAll iterate over all the configured repository issues (comments) and ensure the creation
// of the missing issues / comments / label events / title changes ...
func (gi *giteaImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	gi.iterator = iterator.NewIterator(ctx, gi.client, 10, gi.conf[confKeyOwner], gi.conf[confKeyProject], defaultTimeout, since)
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(gi.out)

		// Loop over all matching issues
		for gi.iterator.NextIssue() {
			issue := gi.iterator.IssueValue()

			// create issue
			b, err := gi.ensureIssue(ctx, repo, issue)
			if err != nil {
				gi.reportError(ctx, err, "issue creation")
				return
			}

			// Loop over all comments
			// TODO: make this a goroutine so we can import issues and comments in parallel?
			// The Github/Gitlab backends already do this. But maybe that's premature
			// optimization.
			for gi.iterator.NextComment() {
				if err = gi.iterator.Error(); err != nil {
					gi.reportError(ctx, err, "fetch comments")
					return
				}
				if err = gi.importComment(ctx, repo, b, gi.iterator.CommentValue()); err != nil {
					gi.reportError(ctx, err, "comment creation")
					return;
				}
			}

			// Loop over all label events
			for gi.iterator.NextLabel() {
				if err = gi.iterator.Error(); err != nil {
					gi.reportError(ctx, err, "fetch labels")
					return
				}
				if err = gi.importLabel(ctx, repo, b, gi.iterator.LabelValue()); err != nil {
					gi.reportError(ctx, err, "update label")
					return
				}
			}

			// Update issue title and description

			if !b.NeedCommit() {
				gi.sendImportResult(ctx, core.NewImportNothing(b.Id(), "no imported operation"))
			} else if err := b.Commit(); err != nil {
				// commit bug state
				gi.reportError(ctx, err, "bug commit")
				return
			}
		}

		if err := gi.iterator.Error(); err != nil {
			gi.reportError(ctx, err, "fetching issues")
		}
	}()

	return out, nil
}

func (gi *giteaImporter) reportError(ctx context.Context, err error, when string) {
	gi.sendImportResult(ctx, core.NewImportError(fmt.Errorf("%s: %v", when, err), ""))
}

func (gi *giteaImporter) sendImportResult(ctx context.Context, result core.ImportResult) {
	select {
	case gi.out <- result:
	// Handle cancellation.
	case <- ctx.Done():
	}
}

func (gi *giteaImporter) importComment(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, comment *gitea.Comment) error {
	commentID := strconv.FormatInt(comment.ID, 10)

	// Check if we've already imported this comment.
	// This isn't as slow as it looks, we're only iterating events on the current issue.
	for _, op := range bug.Snapshot().Operations {
		id, ok := op.GetMetadata(metaKeyGiteaCommentID)
		if ok && id == commentID {
			return nil
		}
	}

	author, err := gi.ensurePerson(ctx, repo, comment.Poster)
	if err != nil {
		return err
	}

	// Needed for deduplication.
	metadata := map[string]string{metaKeyGiteaCommentID: commentID}
	_, _, err = bug.AddCommentRaw(
		author,
		comment.Created.Unix(),
		comment.Body,
		// TODO: add attachments
		make([]repository.Hash, 0),
		metadata,
	)
	return err
}

func (gi *giteaImporter) importLabel(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, event *iterator.LabelEvent) error {
	labelID := strconv.FormatInt(event.Label.ID, 10)
	author, err := gi.ensurePerson(ctx, repo, event.Poster)
	if err != nil {
		return err
	}

	var added, removed []string
	switch event.Kind {
	case iterator.LabelAdded:
		added = []string{event.Label.Name}
	case iterator.LabelRemoved:
		removed = []string{event.Label.Name}
	}

	_, err = bug.ForceChangeLabelsRaw(
		author,
		event.UpdatedAt.Unix(),
		added,
		removed,
		map[string]string{
			metaKeyGiteaID: labelID,
		},
	)
	return err
}

func (gi *giteaImporter) ensureIssue(ctx context.Context, repo *cache.RepoCache, issue *gitea.Issue) (*cache.BugCache, error) {
	author, err := gi.ensurePerson(ctx, repo, issue.Poster)
	if err != nil {
		return nil, err
	}

	giteaID := strconv.FormatInt(issue.Index, 10)

	// resolve bug
	b, err := repo.Bugs().ResolveMatcher(func(excerpt *cache.BugExcerpt) bool {
		return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
			excerpt.CreateMetadata[metaKeyGiteaID] == giteaID &&
			excerpt.CreateMetadata[metaKeyGiteaBaseURL] == gi.conf[confKeyBaseURL] &&
			excerpt.CreateMetadata[metaKeyGiteaOwner] == gi.conf[confKeyOwner] &&
			excerpt.CreateMetadata[metaKeyGiteaProject] == gi.conf[confKeyProject]
	})
	if err == nil {
		return b, nil
	}
	if !entity.IsErrNotFound(err) {
		return nil, err
	}

	// if bug was never imported, create bug
	b, _, err = repo.Bugs().NewRaw(
		author,
		issue.Created.Unix(),
		text.CleanupOneLine(issue.Title),
		text.Cleanup(issue.Body),
		nil,
		map[string]string{
			core.MetaKeyOrigin:  target,
			metaKeyGiteaID:      giteaID,
			metaKeyGiteaOwner:   gi.conf[confKeyOwner],
			metaKeyGiteaProject: gi.conf[confKeyProject],
			metaKeyGiteaBaseURL: gi.conf[confKeyBaseURL],
		},
	)

	if err != nil {
		return nil, err
	}

	gi.sendImportResult(ctx, core.NewImportBug(b.Id()))

	return b, nil
}

func (gi *giteaImporter) ensurePerson(ctx context.Context, repo *cache.RepoCache, poster *gitea.User) (*cache.IdentityCache, error) {
	if poster == nil {
		return gi.deletedIdentity(ctx, repo)
	}

	username := poster.UserName

	// Look first in the cache
	i, err := getCachedIdentity(repo, username)
	if i != nil || err != nil {
		return i, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	user, resp, err := gi.client.Users.GetUserInfo(ctx, username)
	if resp != nil && resp.StatusCode == 404 {
		user = &gitea.User { FullName: username, UserName: username }
	} else if err != nil {
		return nil, err
	}

	i, err = repo.Identities().NewRaw(
		user.FullName,
		user.Email,
		user.UserName,
		user.AvatarURL,
		nil,
		map[string]string{
			// because Gitea
			metaKeyGiteaLogin: user.UserName,
		},
	)
	if err != nil {
		return nil, err
	}

	gi.sendImportResult(ctx, core.NewImportIdentity(i.Id()))
	return i, nil
}

func getCachedIdentity(repo *cache.RepoCache, loginName string) (*cache.IdentityCache, error) {
	i, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyGiteaLogin, loginName)
	if entity.IsErrNotFound(err) {
		return nil, nil
	}
	return i, err
}

func (gi *giteaImporter) deletedIdentity(ctx context.Context, repo *cache.RepoCache) (*cache.IdentityCache, error) {
	i, err := getCachedIdentity(repo, DeletedIdentity)
	if i != nil || err != nil {
		return i, err
	}
	i, err = repo.Identities().NewRaw(
		"Ghost",
		"ghost@example.com",
		DeletedIdentity,
		"",
		nil,
		map[string]string{
			// because Gitea
			metaKeyGiteaLogin: DeletedIdentity,
		},
	)
	if err != nil {
		return nil, err
	}
	gi.sendImportResult(ctx, core.NewImportIdentity(i.Id()))
	return i, nil
}
