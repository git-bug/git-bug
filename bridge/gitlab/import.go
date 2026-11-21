package gitlab

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/gitlab-org/api/client-go"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
)

// gitlabImporter implement the Importer interface
type gitlabImporter struct {
	conf core.Configuration

	// default client
	client *gitlab.Client

	// send only channel
	out chan<- core.ImportResult

	// maps remote timeline event ID -> matched exported operation entry key
	matchedExported map[string]string
}

func (gi *gitlabImporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	gi.conf = conf

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyGitlabBaseUrl]),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrMissingIdentityToken
	}

	gi.client, err = buildClient(conf[confKeyGitlabBaseUrl], creds[0].(*auth.Token))
	if err != nil {
		return err
	}

	return nil
}

// ImportAll iterate over all the configured repository issues (notes) and ensure the creation
// of the missing issues / comments / label events / title changes ...
func (gi *gitlabImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	gi.matchedExported = make(map[string]string)
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(out)

		for result := range Issues(ctx, gi.client, gi.conf[confKeyProjectID], since) {
			if result.Err != nil {
				err := fmt.Errorf("listing issues: %v", result.Err)
				out <- core.NewImportError(err, "")
				return
			}
			issue := result.Issue

			b, err := gi.ensureIssue(ctx, repo, issue)
			if err != nil {
				err := fmt.Errorf("issue creation: %v", err)
				out <- core.NewImportError(err, "")
				return
			}

			issueEvents := SortedEvents(
				Notes(ctx, gi.client, issue),
				LabelEvents(ctx, gi.client, issue),
				StateEvents(ctx, gi.client, issue),
			)

			for e := range issueEvents {
				if e, ok := e.(ErrorEvent); ok {
					out <- core.NewImportError(e.Err, "")
					continue
				}
				if err := gi.ensureIssueEvent(ctx, repo, b, issue, e); err != nil {
					err := fmt.Errorf("unable to create issue event: %v", err)
					out <- core.NewImportError(err, entity.Id(e.ID()))
				}
			}

			if !b.NeedCommit() {
				out <- core.NewImportNothing(b.Id(), "no imported operation")
			} else if err := b.Commit(ctx); err != nil {
				// commit bug state
				err := fmt.Errorf("bug commit: %v", err)
				out <- core.NewImportError(err, "")
				return
			}
		}
	}()

	return out, nil
}

func (gi *gitlabImporter) ensureIssue(ctx context.Context, repo *cache.RepoCache, issue *gitlab.Issue) (*cache.BugCache, error) {
	// ensure issue author
	author, err := gi.ensurePerson(ctx, repo, issue.Author.ID)
	if err != nil {
		return nil, err
	}

	// resolve bug
	b, err := repo.Bugs().ResolveMatcher(ctx, func(excerpt *cache.BugExcerpt) bool {
		return excerpt.CreateMetadata[metaKeyGitlabId] == fmt.Sprintf("%d", issue.IID) &&
			excerpt.CreateMetadata[metaKeyGitlabBaseUrl] == gi.conf[confKeyGitlabBaseUrl] &&
			excerpt.CreateMetadata[metaKeyGitlabProject] == gi.conf[confKeyProjectID]
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
		issue.CreatedAt.Unix(),
		text.CleanupOneLine(issue.Title),
		text.Cleanup(issue.Description),
		nil,
		map[string]string{
			core.MetaKeyOrigin:   target,
			metaKeyGitlabId:      fmt.Sprintf("%d", issue.IID),
			metaKeyGitlabUrl:     issue.WebURL,
			metaKeyGitlabProject: gi.conf[confKeyProjectID],
			metaKeyGitlabBaseUrl: gi.conf[confKeyGitlabBaseUrl],
		},
	)

	if err != nil {
		return nil, err
	}

	// importing a new bug
	gi.out <- core.NewImportBug(b.Id())

	return b, nil
}

func (gi *gitlabImporter) ensureIssueEvent(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, issue *gitlab.Issue, event Event) error {
	var id entity.Id
	var errResolve error
	if event.Kind() == EventComment {
		id, errResolve = resolveCommentTargetWithMetadata(b, metaKeyGitlabId, event.ID())
	} else {
		id, errResolve = b.ResolveOperationWithMetadata(metaKeyGitlabId, event.ID())
	}
	if errResolve != nil && errResolve != cache.ErrNoMatchingOp {
		return errResolve
	}

	// ensure issue author
	author, err := gi.ensurePerson(ctx, repo, event.UserID())
	if err != nil {
		return err
	}

	switch event.Kind() {
	case EventClosed:
		if errResolve == nil || gi.matchExportedStatusChange(b, event.ID(), common.ClosedStatus) {
			return nil
		}

		op, err := b.CloseRaw(
			author,
			event.CreatedAt().Unix(),
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)

		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(b.Id(), op.Id())

	case EventReopened:
		if errResolve == nil || gi.matchExportedStatusChange(b, event.ID(), common.OpenStatus) {
			return nil
		}

		op, err := b.OpenRaw(
			author,
			event.CreatedAt().Unix(),
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportStatusChange(b.Id(), op.Id())

	case EventDescriptionChanged:
		firstComment := b.Snapshot().Comments[0]
		// since gitlab doesn't provide the issue history
		// we should check for "changed the description" notes and compare issue texts
		// TODO: Check only one time and ignore next 'description change' within one issue
		cleanedDesc := text.Cleanup(issue.Description)
		if errResolve == cache.ErrNoMatchingOp && cleanedDesc != firstComment.Message {
			// comment edition
			op, err := b.EditCommentRaw(
				author,
				event.(NoteEvent).UpdatedAt.Unix(),
				firstComment.CombinedId(),
				cleanedDesc,
				map[string]string{
					metaKeyGitlabId: event.ID(),
				},
			)
			if err != nil {
				return err
			}

			gi.out <- core.NewImportTitleEdition(b.Id(), op.Id())
		}

	case EventComment:
		cleanText := text.Cleanup(event.(NoteEvent).Body)

		// if we didn't import the comment
		if errResolve == cache.ErrNoMatchingOp {

			// add comment operation
			commentId, _, err := b.AddCommentRaw(
				author,
				event.CreatedAt().Unix(),
				cleanText,
				nil,
				map[string]string{
					metaKeyGitlabId: event.ID(),
				},
			)
			if err != nil {
				return err
			}
			gi.out <- core.NewImportComment(b.Id(), commentId)
			return nil
		}

		// if comment was already exported

		// search for last comment update
		comment, err := b.Snapshot().SearchCommentByOpId(id)
		if err != nil {
			return err
		}

		// compare local bug comment with the new event body
		if comment.Message != cleanText {
			// comment edition
			_, err := b.EditCommentRaw(
				author,
				event.(NoteEvent).UpdatedAt.Unix(),
				comment.CombinedId(),
				cleanText,
				nil,
			)

			if err != nil {
				return err
			}
			gi.out <- core.NewImportCommentEdition(b.Id(), comment.CombinedId())
		}

		return nil

	case EventTitleChanged:
		// title change events are given new notes
		if errResolve == nil {
			return nil
		}

		title, err := event.(NoteEvent).Title()
		if err != nil {
			return err
		}
		if gi.matchExportedTitleChange(b, event.ID(), title) {
			return nil
		}

		op, err := b.SetTitleRaw(
			author,
			event.CreatedAt().Unix(),
			title,
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportTitleEdition(b.Id(), op.Id())

	case EventAddLabel:
		labelName := event.(LabelEvent).Label.Name
		if errResolve == nil || gi.matchExportedLabelChange(b, event.ID(), labelName, true) {
			return nil
		}
		_, err = b.ForceChangeLabelsRaw(
			author,
			event.CreatedAt().Unix(),
			[]string{labelName},
			nil,
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)
		return err

	case EventRemoveLabel:
		labelName := event.(LabelEvent).Label.Name
		if errResolve == nil || gi.matchExportedLabelChange(b, event.ID(), labelName, false) {
			return nil
		}
		_, err = b.ForceChangeLabelsRaw(
			author,
			event.CreatedAt().Unix(),
			nil,
			[]string{labelName},
			map[string]string{
				metaKeyGitlabId: event.ID(),
			},
		)
		return err

	case EventAssigned,
		EventUnassigned,
		EventChangedMilestone,
		EventRemovedMilestone,
		EventChangedDuedate,
		EventRemovedDuedate,
		EventLocked,
		EventUnlocked,
		EventMentionedInIssue,
		EventMentionedInMergeRequest,
		EventMentionedInCommit:

		return nil

	default:
		return fmt.Errorf("unexpected event: %v", event)
	}

	return nil
}

func (gi *gitlabImporter) ensurePerson(ctx context.Context, repo *cache.RepoCache, id int64) (*cache.IdentityCache, error) {
	// Look first in the cache
	i, err := repo.Identities().ResolveIdentityImmutableMetadata(ctx, metaKeyGitlabId, strconv.FormatInt(id, 10))
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}

	user, _, err := gi.client.Users.GetUser(id, gitlab.GetUsersOptions{})
	if err != nil {
		return nil, err
	}

	i, err = repo.Identities().NewRaw(
		ctx,
		user.Name,
		user.PublicEmail,
		user.Username,
		user.AvatarURL,
		nil,
		map[string]string{
			// because Gitlab
			metaKeyGitlabId:    strconv.FormatInt(id, 10),
			metaKeyGitlabLogin: user.Username,
		},
	)
	if err != nil {
		return nil, err
	}

	gi.out <- core.NewImportIdentity(i.Id())
	return i, nil
}

func resolveCommentTargetWithMetadata(b *cache.BugCache, key, value string) (entity.Id, error) {
	matching := make([]entity.Id, 0, 1)
	for _, op := range b.Snapshot().AllOperations() {
		switch op.(type) {
		case *bug.CreateOperation, *bug.AddCommentOperation:
			if opVal, ok := op.GetMetadata(key); ok && opVal == value {
				matching = append(matching, op.Id())
			}
		}
	}
	if len(matching) == 0 {
		return "", cache.ErrNoMatchingOp
	}
	if len(matching) > 1 {
		return "", entity.NewErrMultipleMatch("operation", matching)
	}
	return matching[0], nil
}

func (gi *gitlabImporter) isOpEntryClaimed(entryKey, eventKey string) bool {
	if gi.matchedExported == nil {
		gi.matchedExported = make(map[string]string)
	}
	for evKey, key := range gi.matchedExported {
		if key == entryKey && evKey != eventKey {
			return true
		}
	}
	return false
}

func (gi *gitlabImporter) matchExportedStatusChange(b *cache.BugCache, eventID string, status common.Status) bool {
	eventKey := fmt.Sprintf("%s:status:%s", b.Id(), eventID)
	if gi.matchedExported != nil {
		if _, ok := gi.matchedExported[eventKey]; ok {
			return true
		}
	}
	snap := b.Snapshot()
	issueID, ok := snap.GetCreateMetadata(metaKeyGitlabId)
	if !ok {
		return false
	}
	for _, op := range snap.Operations {
		statusOp, ok := op.(*bug.SetStatusOperation)
		if !ok || statusOp.Status != status {
			continue
		}
		if id, ok := statusOp.GetMetadata(metaKeyGitlabId); !ok || id != issueID {
			continue
		}
		entryKey := statusOp.Id().String()
		if gi.isOpEntryClaimed(entryKey, eventKey) {
			continue
		}
		gi.matchedExported[eventKey] = entryKey
		return true
	}
	return false
}

func (gi *gitlabImporter) matchExportedTitleChange(b *cache.BugCache, eventID, title string) bool {
	eventKey := fmt.Sprintf("%s:title:%s", b.Id(), eventID)
	if gi.matchedExported != nil {
		if _, ok := gi.matchedExported[eventKey]; ok {
			return true
		}
	}
	snap := b.Snapshot()
	issueID, ok := snap.GetCreateMetadata(metaKeyGitlabId)
	if !ok {
		return false
	}
	for _, op := range snap.Operations {
		titleOp, ok := op.(*bug.SetTitleOperation)
		if !ok || titleOp.Title != title {
			continue
		}
		if id, ok := titleOp.GetMetadata(metaKeyGitlabId); !ok || id != issueID {
			continue
		}
		entryKey := titleOp.Id().String()
		if gi.isOpEntryClaimed(entryKey, eventKey) {
			continue
		}
		gi.matchedExported[eventKey] = entryKey
		return true
	}
	return false
}

func (gi *gitlabImporter) matchExportedLabelChange(b *cache.BugCache, eventID, label string, added bool) bool {
	eventKey := fmt.Sprintf("%s:label:%s", b.Id(), eventID)
	if gi.matchedExported != nil {
		if _, ok := gi.matchedExported[eventKey]; ok {
			return true
		}
	}
	snap := b.Snapshot()
	issueID, ok := snap.GetCreateMetadata(metaKeyGitlabId)
	if !ok {
		return false
	}
	for _, op := range snap.Operations {
		labelOp, ok := op.(*bug.LabelChangeOperation)
		if !ok {
			continue
		}
		if id, ok := labelOp.GetMetadata(metaKeyGitlabId); !ok || id != issueID {
			continue
		}
		hasLabel := false
		if added {
			for _, l := range labelOp.Added {
				if string(l) == label {
					hasLabel = true
					break
				}
			}
		} else {
			for _, l := range labelOp.Removed {
				if string(l) == label {
					hasLabel = true
					break
				}
			}
		}
		if !hasLabel {
			continue
		}
		entryKey := fmt.Sprintf("%s:%t:%s", labelOp.Id().String(), added, label)
		if gi.isOpEntryClaimed(entryKey, eventKey) {
			continue
		}
		gi.matchedExported[eventKey] = entryKey
		return true
	}
	return false
}
