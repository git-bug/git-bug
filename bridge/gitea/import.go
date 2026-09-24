package gitea

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	gitea "gitea.dev/sdk"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/bridge/gitea/iterator"
	"github.com/git-bug/git-bug/cache"
	bugpkg "github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
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

			// create a record to point to
			b, err := gi.ensureIssue(ctx, repo, issue)
			if err != nil {
				gi.reportError(ctx, err, "issue creation")
				return
			}

			// update status/title/body
			if err = gi.updateIssue(ctx, repo, b, issue); err != nil {
				gi.reportBugError(ctx, err, "issue update", b.Id())
				return
			}

			// Loop over all events
			// TODO: make this a goroutine so we can import issues and events in parallel?
			// The Github/Gitlab backends already do this. But maybe that's premature
			// optimization.
			for gi.iterator.NextEvent() {
				if err = gi.importEvent(ctx, repo, b, gi.iterator.EventValue()); err != nil {
					gi.reportBugError(ctx, err, "import timeline event", b.Id())
					return
				}
			}
			if err = gi.iterator.Error(); err != nil {
				gi.reportBugError(ctx, err, "fetch timeline", b.Id())
				return
			}

			if err = gi.reconcileLabels(ctx, repo, b, issue); err != nil {
				gi.reportBugError(ctx, err, "reconcile labels", b.Id())
				return
			}

			if !b.NeedCommit() {
				gi.sendImportResult(ctx, core.NewImportNothing(b.Id(), "no imported operation"))
			} else if err := b.Commit(); err != nil {
				// commit bug state
				gi.reportBugError(ctx, err, "bug commit", b.Id())
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
	gi.reportBugError(ctx, err, when, "")
}

// reportBugError reports an error that concerns an already-created bug.
func (gi *giteaImporter) reportBugError(ctx context.Context, err error, when string, id entity.Id) {
	gi.sendImportResult(ctx, core.NewImportError(fmt.Errorf("%s: %v", when, err), id))
}

func (gi *giteaImporter) sendImportResult(ctx context.Context, result core.ImportResult) {
	select {
	case gi.out <- result:
	// Handle cancellation.
	case <-ctx.Done():
	}
}

func (gi *giteaImporter) updateIssue(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, issue *gitea.Issue) error {
	switch issue.State {
	case gitea.StateOpen:
		// b.OpenRaw()
	case gitea.StateClosed:
		// b.CloseRaw()
	}
	return nil
}

func (gi *giteaImporter) importEvent(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, event iterator.TimelineEvent) error {
	switch e := event.(type) {
	case *iterator.LabelEvent:
		return gi.importLabel(ctx, repo, bug, e)
	case *iterator.CommentEvent:
		return gi.importComment(ctx, repo, bug, e)
	case *iterator.RenameEvent:
		return gi.importRename(ctx, repo, bug, e)
	case *iterator.StatusEvent:
		return gi.importStatus(ctx, repo, bug, e)
	}
	return fmt.Errorf("unsupported timeline event %T", event)
}

// timelineEventKey identifies a timeline event across imports. Gitea event IDs
// are unique, but the type and time are included so events without an ID
// (as some servers and test fixtures send) still get distinct keys.
func timelineEventKey(kind string, id int64, at time.Time) string {
	return fmt.Sprintf("%s:%d:%d", kind, id, at.Unix())
}

// alreadyImported reports whether an operation carrying the given timeline
// event key exists on the bug.
func alreadyImported(bug *cache.BugCache, key string) bool {
	for _, op := range bug.Snapshot().Operations {
		if v, ok := op.GetMetadata(metaKeyGiteaEvent); ok && v == key {
			return true
		}
	}
	return false
}

func (gi *giteaImporter) importComment(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, remoteComment *iterator.CommentEvent) error {
	author, err := gi.ensurePerson(ctx, repo, remoteComment.Poster)
	if err != nil {
		return err
	}

	// Needed for deduplication.
	giteaId := strconv.FormatInt(remoteComment.ID, 10)
	metadata := map[string]string{metaKeyGiteaCommentID: giteaId}

	// Check if we've already imported this comment. Only comment creations
	// count: edits carry the same metadata but aren't comments themselves.
	// This isn't as slow as it looks, we're only iterating events on the current issue.
	var op dag.Operation
	for _, candidate := range bug.Snapshot().Operations {
		if _, isAdd := candidate.(*bugpkg.AddCommentOperation); !isAdd {
			continue
		}
		if existingId, ok := candidate.GetMetadata(metaKeyGiteaCommentID); ok && existingId == giteaId {
			op = candidate
			break
		}
	}

	// Check if we're creating a new comment or just updating an existing one.
	// Forgejo doesn't have an API for this unfortunately, so we're stuck with comparing
	// the body text. Note this means we might miss intermediate edits.
	if op != nil {
		localComment, err := bug.Snapshot().SearchCommentByOpId(op.Id())
		if err != nil {
			return err
		}
		if localComment == nil {
			panic("found bug by metadata, but not by op ID?")
		}

		if localComment.Message == remoteComment.Body {
			return nil
		}

		_, err = bug.EditCommentRaw(
			author,
			remoteComment.Updated.Unix(),
			localComment.CombinedId(),
			remoteComment.Body,
			metadata,
		)

		return err
	}

	_, _, err = bug.AddCommentRaw(
		author,
		remoteComment.Created.Unix(),
		remoteComment.Body,
		// NOTE: attachments are not supported (none of the either backends support them either)
		make([]repository.Hash, 0),
		metadata,
	)
	return err
}

func (gi *giteaImporter) importLabel(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, event *iterator.LabelEvent) error {
	// Gitea sends no label object when the label has since been deleted.
	// There is no name to apply; reconcileLabels handles the current state.
	if event.Label == nil {
		return nil
	}

	kind := "label-remove"
	if event.Kind == iterator.LabelAdded {
		kind = "label-add"
	}
	key := timelineEventKey(kind, int64(event.ID), event.UpdatedAt)
	if alreadyImported(bug, key) {
		return nil
	}

	// Compare names case-insensitively, so an upstream "Bug" does not end
	// up next to an existing local "bug".
	local, present := findLabelFold(bug, event.Label.Name)
	var added, removed []string
	switch {
	case event.Kind == iterator.LabelAdded && !present:
		added = []string{event.Label.Name}
	case event.Kind == iterator.LabelRemoved && present:
		removed = []string{local}
	default:
		return nil
	}

	author, err := gi.ensurePerson(ctx, repo, event.Poster)
	if err != nil {
		return err
	}

	_, err = bug.ForceChangeLabelsRaw(
		author,
		event.UpdatedAt.Unix(),
		added,
		removed,
		map[string]string{
			metaKeyGiteaID:    strconv.FormatInt(event.Label.ID, 10),
			metaKeyGiteaEvent: key,
		},
	)
	return err
}

// findLabelFold returns the bug's label equal to name ignoring case.
func findLabelFold(bug *cache.BugCache, name string) (string, bool) {
	for _, l := range bug.Snapshot().Labels {
		if strings.EqualFold(string(l), name) {
			return string(l), true
		}
	}
	return "", false
}

// reconcileLabels brings the bug's labels in line with the issue's current
// upstream labels. Timeline events report labels under their current name, so
// replaying them misses renames; this catches renames and any other drift.
//
// Only labels that a Gitea import added are removed, so labels added locally
// in git-bug survive. A nil label list means the server didn't report labels
// (Gitea always sends an array), so nothing is reconciled.
func (gi *giteaImporter) reconcileLabels(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, issue *gitea.Issue) error {
	if issue.Labels == nil {
		return nil
	}

	imported := map[string]bool{}
	for _, op := range bug.Snapshot().Operations {
		labelOp, ok := op.(*bugpkg.LabelChangeOperation)
		if !ok {
			continue
		}
		if _, fromGitea := op.GetMetadata(metaKeyGiteaID); !fromGitea {
			continue
		}
		for _, l := range labelOp.Added {
			imported[strings.ToLower(string(l))] = true
		}
	}

	upstream := map[string]bool{}
	var added []string
	for _, l := range issue.Labels {
		if l == nil {
			continue
		}
		upstream[strings.ToLower(l.Name)] = true
		if _, present := findLabelFold(bug, l.Name); !present {
			added = append(added, l.Name)
		}
	}
	var removed []string
	for _, l := range bug.Snapshot().Labels {
		folded := strings.ToLower(string(l))
		if !upstream[folded] && imported[folded] {
			removed = append(removed, string(l))
		}
	}
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}

	// The timeline doesn't say who caused the drift (for example a label
	// rename), so attribute the change to the issue author.
	author, err := gi.ensurePerson(ctx, repo, issue.Poster)
	if err != nil {
		return err
	}
	at := issue.Updated
	if at.IsZero() {
		at = time.Now()
	}
	_, err = bug.ForceChangeLabelsRaw(author, at.Unix(), added, removed,
		map[string]string{metaKeyGiteaID: "reconcile"})
	return err
}

func (gi *giteaImporter) importRename(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, rename *iterator.RenameEvent) error {
	key := timelineEventKey("change_title", rename.ID, rename.Updated)
	if alreadyImported(bug, key) {
		return nil
	}

	author, err := gi.ensurePerson(ctx, repo, rename.Poster)
	if err != nil {
		return err
	}

	_, err = bug.SetTitleRaw(
		author,
		rename.Updated.Unix(),
		cleanTitle(rename.NewName),
		map[string]string{metaKeyGiteaEvent: key},
	)
	return err
}

func (gi *giteaImporter) importStatus(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, event *iterator.StatusEvent) error {
	kind := "reopen"
	if event.Closed {
		kind = "close"
	}
	key := timelineEventKey(kind, event.ID, event.Created)
	if alreadyImported(bug, key) {
		return nil
	}

	author, err := gi.ensurePerson(ctx, repo, event.Poster)
	if err != nil {
		return err
	}

	metadata := map[string]string{metaKeyGiteaEvent: key}
	if event.Closed {
		_, err = bug.CloseRaw(author, event.Created.Unix(), metadata)
	} else {
		_, err = bug.OpenRaw(author, event.Created.Unix(), metadata)
	}
	return err
}

// EmptyTitlePlaceholder replaces titles that are empty after cleanup, since
// git-bug rejects empty titles. Matches the GitHub bridge.
const EmptyTitlePlaceholder = "<empty string>"

func cleanTitle(title string) string {
	title = text.CleanupOneLine(title)
	if text.Empty(title) {
		return EmptyTitlePlaceholder
	}
	return title
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
		cleanTitle(issue.Title),
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
		user = &gitea.User{FullName: username, UserName: username}
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
