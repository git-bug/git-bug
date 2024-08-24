package launchpad

import (
	"context"
	"fmt"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
)

type launchpadImporter struct {
	conf core.Configuration
}

func (li *launchpadImporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	li.conf = conf
	return nil
}

func (li *launchpadImporter) ensurePerson(repo *cache.RepoCache, owner LPPerson) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyLaunchpadLogin, owner.Login)
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}

	return repo.Identities().NewRaw(
		owner.Name,
		"",
		owner.Login,
		"",
		nil,
		map[string]string{
			metaKeyLaunchpadLogin: owner.Login,
		},
	)
}

func (li *launchpadImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	out := make(chan core.ImportResult)
	lpAPI := new(launchpadAPI)

	err := lpAPI.Init()
	if err != nil {
		return nil, err
	}

	lpBugs, err := lpAPI.SearchTasks(ctx, li.conf["project"])
	if err != nil {
		return nil, err
	}

	go func() {
		for _, lpBug := range lpBugs {
			select {
			case <-ctx.Done():
				return
			default:
				lpBugID := fmt.Sprintf("%d", lpBug.ID)
				b, err := repo.Bugs().ResolveMatcher(func(excerpt *cache.BugExcerpt) bool {
					return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
						excerpt.CreateMetadata[metaKeyLaunchpadID] == lpBugID
				})
				if err != nil && !entity.IsErrNotFound(err) {
					out <- core.NewImportError(err, entity.Id(lpBugID))
					return
				}

				owner, err := li.ensurePerson(repo, lpBug.Owner)
				if err != nil {
					out <- core.NewImportError(err, entity.Id(lpBugID))
					return
				}

				if entity.IsErrNotFound(err) {
					createdAt, _ := time.Parse(time.RFC3339, lpBug.CreatedAt)
					b, _, err = repo.Bugs().NewRaw(
						owner,
						createdAt.Unix(),
						text.CleanupOneLine(lpBug.Title),
						text.Cleanup(lpBug.Description),
						nil,
						map[string]string{
							core.MetaKeyOrigin: target,
							metaKeyLaunchpadID: lpBugID,
						},
					)
					if err != nil {
						out <- core.NewImportError(err, entity.Id(lpBugID))
						return
					}

					out <- core.NewImportBug(b.Id())

				}

				/* Handle messages */
				if len(lpBug.Messages) == 0 {
					return
				}

				// The Launchpad API returns the bug description as the first
				// comment, so skip it.
				for _, lpMessage := range lpBug.Messages[1:] {
					_, err := b.ResolveOperationWithMetadata(metaKeyLaunchpadID, lpMessage.ID)
					if err != nil && err != cache.ErrNoMatchingOp {
						out <- core.NewImportError(err, entity.Id(lpMessage.ID))
						return
					}

					// If this comment already exists, we are probably
					// updating an existing bug. We do not want to duplicate
					// the comments, so let us just skip this one.
					// TODO: Can Launchpad comments be edited?
					if err == nil {
						continue
					}

					owner, err := li.ensurePerson(repo, lpMessage.Owner)
					if err != nil {
						out <- core.NewImportError(err, "")
						return
					}

					// This is a new comment, we can add it.
					createdAt, _ := time.Parse(time.RFC3339, lpMessage.CreatedAt)
					commentId, _, err := b.AddCommentRaw(
						owner,
						createdAt.Unix(),
						text.Cleanup(lpMessage.Content),
						nil,
						map[string]string{
							metaKeyLaunchpadID: lpMessage.ID,
						})
					if err != nil {
						out <- core.NewImportError(err, b.Id())
						return
					}

					out <- core.NewImportComment(b.Id(), commentId)
				}

				if !b.NeedCommit() {
					out <- core.NewImportNothing(b.Id(), "no imported operation")
				} else if err := b.Commit(); err != nil {
					out <- core.NewImportError(err, "")
					return
				}
			}
		}
	}()

	return out, nil
}
