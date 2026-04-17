package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
)

const EmptyTitlePlaceholder = "<empty string>"

// githubImporter implement the Importer interface
type githubImporter struct {
	conf core.Configuration

	// default client
	client *rateLimitHandlerClient

	// mediator to access the Github API
	mediator *importMediator

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
	if len(creds) <= 0 {
		return ErrMissingIdentityToken
	}
	gi.client = buildClient(creds[0].(*auth.Token))

	return nil
}

// ImportAll iterate over all the configured repository issues and ensure the creation of the
// missing issues / timeline items / edits / label events ...
func (gi *githubImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	gi.mediator = NewImportMediator(ctx, gi.client, gi.conf[confKeyOwner], gi.conf[confKeyProject], since)
	out := make(chan core.ImportResult)
	gi.out = out

	go func() {
		defer close(gi.out)
		var currBug *cache.BugCache
		var currEvent ImportEvent
		var nextEvent ImportEvent
		var err error
		for {
			// An IssueEvent contains the issue in its most recent state. If an issue
			// has at least one issue edit, then the history of the issue edits is
			// represented by IssueEditEvents. That is, the unedited (original) issue
			// might be saved only in the IssueEditEvent following the IssueEvent.
			// Since we replicate the edit history we need to either use the IssueEvent
			// (if there are no edits) or the IssueEvent together with its first
			// IssueEditEvent (if there are edits).
			// Exactly the same is true for comments and comment edits.
			// As a consequence we need to look at the current event and one look ahead
			// event.

			currEvent = nextEvent
			if currEvent == nil {
				currEvent = gi.getEventHandleMsgs()
			}
			if currEvent == nil {
				break
			}
			nextEvent = gi.getEventHandleMsgs()

			switch event := currEvent.(type) {
			case RateLimitingEvent:
				out <- core.NewImportRateLimiting(event.msg)
			case IssueEvent:
				// first: commit what is being held in currBug
				if err = gi.commit(currBug, out); err != nil {
					out <- core.NewImportError(err, "")
					return
				}
				// second: create new issue
				switch next := nextEvent.(type) {
				case IssueEditEvent:
					// consuming and using next event
					nextEvent = nil
					currBug, err = gi.ensureIssue(ctx, repo, &event.issue, &next.userContentEdit)
				default:
					currBug, err = gi.ensureIssue(ctx, repo, &event.issue, nil)
				}
				if err != nil {
					err := fmt.Errorf("issue creation: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case IssueEditEvent:
				err = gi.ensureIssueEdit(ctx, repo, currBug, event.issueId, &event.userContentEdit)
				if err != nil {
					err = fmt.Errorf("issue edit: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case TimelineEvent:
				if next, ok := nextEvent.(CommentEditEvent); ok && event.Typename == "IssueComment" {
					// consuming and using next event
					nextEvent = nil
					err = gi.ensureComment(ctx, repo, currBug, &event.timelineItem.IssueComment, &next.userContentEdit)
				} else {
					err = gi.ensureTimelineItem(ctx, repo, currBug, &event.timelineItem)
				}
				if err != nil {
					err = fmt.Errorf("timeline item creation: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case CommentEditEvent:
				err = gi.ensureCommentEdit(ctx, repo, currBug, event.commentId, &event.userContentEdit)
				if err != nil {
					err = fmt.Errorf("comment edit: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case PrEvent:
				if err = gi.commit(currBug, out); err != nil {
					out <- core.NewImportError(err, "")
					return
				}
				switch next := nextEvent.(type) {
				case PrEditEvent:
					nextEvent = nil
					currBug, err = gi.ensurePR(ctx, repo, &event.pullRequest, &next.userContentEdit)
				default:
					currBug, err = gi.ensurePR(ctx, repo, &event.pullRequest, nil)
				}
				if err != nil {
					err := fmt.Errorf("pr creation: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case PrEditEvent:
				err = gi.ensureIssueEdit(ctx, repo, currBug, event.prId, &event.userContentEdit)
				if err != nil {
					err = fmt.Errorf("pr edit: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			case PrTimelineEvent:
				if next, ok := nextEvent.(CommentEditEvent); ok && event.Typename == "IssueComment" {
					nextEvent = nil
					err = gi.ensureComment(ctx, repo, currBug, &event.IssueComment, &next.userContentEdit)
				} else {
					err = gi.ensurePrTimelineItem(ctx, repo, currBug, &event.prTimelineItem)
				}
				if err != nil {
					err = fmt.Errorf("pr timeline item: %v", err)
					out <- core.NewImportError(err, "")
					return
				}
			default:
				panic("Unknown event type")
			}
		}
		// commit what is being held in currBug before returning
		if err = gi.commit(currBug, out); err != nil {
			out <- core.NewImportError(err, "")
		}
		if err = gi.mediator.Error(); err != nil {
			gi.out <- core.NewImportError(err, "")
		}
	}()

	return out, nil
}

func (gi *githubImporter) getEventHandleMsgs() ImportEvent {
	for {
		// read event from import mediator
		event := gi.mediator.NextImportEvent()
		// consume (and use) all rate limiting events
		if e, ok := event.(RateLimitingEvent); ok {
			gi.out <- core.NewImportRateLimiting(e.msg)
			continue
		}
		return event
	}
}

func (gi *githubImporter) commit(b *cache.BugCache, out chan<- core.ImportResult) error {
	if b == nil {
		return nil
	}
	if !b.NeedCommit() {
		out <- core.NewImportNothing(b.Id(), "no imported operation")
		return nil
	} else if err := b.Commit(); err != nil {
		// commit bug state
		return fmt.Errorf("bug commit: %v", err)
	}
	return nil
}

func (gi *githubImporter) ensureIssue(ctx context.Context, repo *cache.RepoCache, issue *issue, issueEdit *userContentEdit) (*cache.BugCache, error) {
	author, err := gi.ensurePerson(ctx, repo, issue.Author)
	if err != nil {
		return nil, err
	}

	// resolve bug
	b, err := repo.Bugs().ResolveMatcher(func(excerpt *cache.BugExcerpt) bool {
		return excerpt.CreateMetadata[metaKeyGithubUrl] == issue.Url.String() &&
			excerpt.CreateMetadata[metaKeyGithubId] == parseId(issue.Id)
	})
	if err == nil {
		return b, nil
	}
	if !entity.IsErrNotFound(err) {
		return nil, err
	}

	// At Github there exist issues with seemingly empty titles. An example is
	// https://github.com/NixOS/nixpkgs/issues/72730 (here the title is actually
	// a zero width space U+200B).
	// Set title to some non-empty string, since git-bug does not accept empty titles.
	title := text.CleanupOneLine(string(issue.Title))
	if text.Empty(title) {
		title = EmptyTitlePlaceholder
	}

	var textInput string
	if issueEdit != nil {
		// use the first issue edit: it represents the bug creation itself
		textInput = string(*issueEdit.Diff)
	} else {
		// if there are no issue edits then the issue struct holds the bug creation
		textInput = string(issue.Body)
	}

	// create bug
	b, _, err = repo.Bugs().NewRaw(
		author,
		issue.CreatedAt.Unix(),
		text.CleanupOneLine(title), // TODO: this is the *current* title, not the original one
		text.Cleanup(textInput),
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

	return b, nil
}

func (gi *githubImporter) ensureIssueEdit(ctx context.Context, repo *cache.RepoCache, bug *cache.BugCache, ghIssueId githubv4.ID, edit *userContentEdit) error {
	return gi.ensureCommentEdit(ctx, repo, bug, ghIssueId, edit)
}

func (gi *githubImporter) ensurePR(ctx context.Context, repo *cache.RepoCache, pr *pullRequest, prEdit *userContentEdit) (*cache.BugCache, error) {
	author, err := gi.ensurePerson(ctx, repo, pr.Author)
	if err != nil {
		return nil, err
	}

	// resolve bug
	b, err := repo.Bugs().ResolveMatcher(func(excerpt *cache.BugExcerpt) bool {
		return excerpt.CreateMetadata[metaKeyGithubUrl] == pr.Url.String() &&
			excerpt.CreateMetadata[metaKeyGithubId] == parseId(pr.Id)
	})
	if err == nil {
		return b, nil
	}
	if !entity.IsErrNotFound(err) {
		return nil, err
	}

	title := text.CleanupOneLine(string(pr.Title))
	if text.Empty(title) {
		title = EmptyTitlePlaceholder
	}

	var textInput string
	if prEdit != nil {
		textInput = string(*prEdit.Diff)
	} else {
		textInput = string(pr.Body)
	}

	baseRef := "refs/heads/" + string(pr.BaseRefName)
	headRef := "refs/heads/" + string(pr.HeadRefName)
	headCommit := string(pr.HeadRefOid)

	b, _, err = repo.Bugs().NewPRRaw(
		author,
		pr.CreatedAt.Unix(),
		text.CleanupOneLine(title),
		text.Cleanup(textInput),
		baseRef,
		headRef,
		headCommit,
		bool(pr.IsDraft),
		nil,
		map[string]string{
			core.MetaKeyOrigin: target,
			metaKeyGithubId:    parseId(pr.Id),
			metaKeyGithubUrl:   pr.Url.String(),
		})
	if err != nil {
		return nil, err
	}

	// If merged or closed at import time, record the terminal state.
	if bool(pr.Merged) && pr.MergeCommit != nil {
		_, err := b.MergeRaw(author, pr.CreatedAt.Unix(), string(pr.MergeCommit.Oid), nil)
		if err != nil {
			return nil, err
		}
	} else if bool(pr.Closed) && !bool(pr.Merged) {
		_, err := b.CloseRaw(author, pr.CreatedAt.Unix(), nil)
		if err != nil {
			return nil, err
		}
	}

	gi.out <- core.NewImportBug(b.Id())
	return b, nil
}

// ensurePrTimelineItem handles a PR timeline event: issue-shared items reuse
// the issue handlers; PR-only items (MergedEvent, ReadyForReviewEvent,
// ConvertToDraftEvent) get dedicated handling.
func (gi *githubImporter) ensurePrTimelineItem(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, item *prTimelineItem) error {
	switch item.Typename {
	case "IssueComment", "LabeledEvent", "UnlabeledEvent", "ClosedEvent", "ReopenedEvent", "RenamedTitleEvent":
		// Forward to the issue timeline handler via a shim. These events share
		// the same payload on issues and PRs in GitHub's data model.
		ti := timelineItem{
			Typename:          item.Typename,
			IssueComment:      item.IssueComment,
			LabeledEvent:      item.LabeledEvent,
			UnlabeledEvent:    item.UnlabeledEvent,
			ClosedEvent:       item.ClosedEvent,
			ReopenedEvent:     item.ReopenedEvent,
			RenamedTitleEvent: item.RenamedTitleEvent,
		}
		return gi.ensureTimelineItem(ctx, repo, b, &ti)

	case "MergedEvent":
		id := parseId(item.MergedEvent.Id)
		if _, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id); err == nil {
			return nil
		} else if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(ctx, repo, item.MergedEvent.Actor)
		if err != nil {
			return err
		}
		commitHash := ""
		if item.MergedEvent.Commit != nil {
			commitHash = string(item.MergedEvent.Commit.Oid)
		}
		if commitHash == "" {
			// Merge commit missing is unusual but possible; skip to avoid
			// creating an invalid operation (Merge validates commit != "").
			return nil
		}
		op, err := b.MergeRaw(
			author,
			item.MergedEvent.CreatedAt.Unix(),
			commitHash,
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}
		gi.out <- core.NewImportStatusChange(b.Id(), op.Id())
		return nil

	case "ReadyForReviewEvent":
		id := parseId(item.ReadyForReviewEvent.Id)
		if _, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id); err == nil {
			return nil
		} else if err != cache.ErrNoMatchingOp {
			return err
		}
		author, err := gi.ensurePerson(ctx, repo, item.ReadyForReviewEvent.Actor)
		if err != nil {
			return err
		}
		op, err := b.OpenRaw(
			author,
			item.ReadyForReviewEvent.CreatedAt.Unix(),
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}
		gi.out <- core.NewImportStatusChange(b.Id(), op.Id())
		return nil

	case "ConvertToDraftEvent":
		// git-bug doesn't have a SetDraft op; converting back to draft is rare
		// on GitHub and carries no new information for v1. Skip silently.
		return nil

	case "PullRequestReview":
		return gi.ensureReview(ctx, repo, b, &item.PullRequestReview)
	}
	return nil
}

// ensureReview creates an AddReviewOperation and any nested
// AddReviewCommentOperations for a GitHub PullRequestReview.
func (gi *githubImporter) ensureReview(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, review *pullRequestReview) error {
	id := parseId(review.Id)

	author, err := gi.ensurePerson(ctx, repo, review.Author)
	if err != nil {
		return err
	}

	reviewOpId, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id)
	if err != nil && err != cache.ErrNoMatchingOp {
		return err
	}

	var commitHash string
	if review.Commit != nil {
		commitHash = string(review.Commit.Oid)
	}
	if commitHash == "" {
		// Reviews must anchor to a commit; skip if GitHub returns nil (unusual
		// but possible for very old or deleted branches).
		return nil
	}

	if err == cache.ErrNoMatchingOp {
		state := mapReviewState(review.State)
		op, newErr := b.AddReviewRaw(
			author,
			review.CreatedAt.Unix(),
			state,
			text.Cleanup(string(review.Body)),
			commitHash,
			map[string]string{metaKeyGithubId: id},
		)
		if newErr != nil {
			return newErr
		}
		reviewOpId = op.Id()
		gi.out <- core.NewImportReview(b.Id(), op.Id())
	}

	// Import inline review comments first, then follow cursor for any extras.
	reviewCombined := entity.CombineIds(b.Id(), reviewOpId)
	for _, rc := range review.Comments.Nodes {
		if err := gi.ensureReviewComment(ctx, repo, b, reviewCombined, &rc); err != nil {
			return err
		}
	}

	if review.Comments.PageInfo.HasNextPage {
		cursor := review.Comments.PageInfo.EndCursor
		for {
			nodes, nextCursor, hasNext := gi.mediator.QueryReviewComments(ctx, review.Id, cursor)
			for i := range nodes {
				if err := gi.ensureReviewComment(ctx, repo, b, reviewCombined, &nodes[i]); err != nil {
					return err
				}
			}
			if !hasNext {
				break
			}
			cursor = nextCursor
		}
	}
	return nil
}

func (gi *githubImporter) ensureReviewComment(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, reviewId entity.CombinedId, c *pullRequestReviewComment) error {
	id := parseId(c.Id)
	if _, err := b.ResolveOperationWithMetadata(metaKeyGithubId, id); err == nil {
		return nil
	} else if err != cache.ErrNoMatchingOp {
		return err
	}

	author, err := gi.ensurePerson(ctx, repo, c.Author)
	if err != nil {
		return err
	}

	var commitHash string
	if c.Commit != nil {
		commitHash = string(c.Commit.Oid)
	}
	if commitHash == "" {
		// Review comments must anchor to a commit. GitHub sometimes returns
		// outdated / null commits for deleted branches; skip those.
		return nil
	}

	startLine := int(c.Line)
	if c.StartLine != nil {
		startLine = int(*c.StartLine)
	}
	endLine := int(c.Line)
	if startLine <= 0 {
		// GitHub returns line=0 for outdated review comments (anchored to a
		// position that no longer exists in the current diff). Skip silently.
		return nil
	}
	if endLine < startLine {
		endLine = startLine
	}

	var replyTo entity.CombinedId
	if c.ReplyTo != nil {
		// The parent review-comment's git-bug combined id, if we've already
		// imported it. We look it up by GitHub id.
		parentOpId, lookupErr := b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(c.ReplyTo.Id))
		if lookupErr == nil {
			replyTo = entity.CombineIds(b.Id(), parentOpId)
		}
		// If parent isn't imported (truncated / deleted), fall back to a
		// top-level comment.
	}

	commentId, _, err := b.AddReviewCommentRaw(
		author,
		c.CreatedAt.Unix(),
		reviewId,
		text.Cleanup(string(c.Body)),
		commitHash,
		string(c.Path),
		startLine,
		endLine,
		replyTo,
		map[string]string{metaKeyGithubId: id},
	)
	if err != nil {
		return err
	}
	gi.out <- core.NewImportReviewComment(b.Id(), commentId)
	return nil
}

// mapReviewState translates GitHub's PullRequestReviewState into git-bug's
// ReviewState.
func mapReviewState(s githubv4.PullRequestReviewState) bug.ReviewState {
	switch s {
	case githubv4.PullRequestReviewStateApproved:
		return bug.ReviewApproved
	case githubv4.PullRequestReviewStateChangesRequested:
		return bug.ReviewChangesRequested
	default:
		// PENDING / COMMENTED / DISMISSED all map to "commented" for now;
		// PENDING reviews shouldn't reach the import (GitHub only exposes
		// submitted reviews), DISMISSED is a state change after submission.
		return bug.ReviewCommented
	}
}

func (gi *githubImporter) ensureTimelineItem(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, item *timelineItem) error {

	switch item.Typename {
	case "IssueComment":
		err := gi.ensureComment(ctx, repo, b, &item.IssueComment, nil)
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
				text.CleanupOneLine(string(item.LabeledEvent.Label.Name)),
			},
			nil,
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportLabelChange(b.Id(), op.Id())
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
				text.CleanupOneLine(string(item.UnlabeledEvent.Label.Name)),
			},
			map[string]string{metaKeyGithubId: id},
		)
		if err != nil {
			return err
		}

		gi.out <- core.NewImportLabelChange(b.Id(), op.Id())
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

		gi.out <- core.NewImportStatusChange(b.Id(), op.Id())
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

		gi.out <- core.NewImportStatusChange(b.Id(), op.Id())
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
		// https://github.com/NixOS/nixpkgs/issues/72730 (here the title is actually
		// a zero width space U+200B).
		// Set title to some non-empty string, since git-bug does not accept empty titles.
		title := text.CleanupOneLine(string(item.RenamedTitleEvent.CurrentTitle))
		if text.Empty(title) {
			title = EmptyTitlePlaceholder
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

		gi.out <- core.NewImportTitleEdition(b.Id(), op.Id())
		return nil
	}

	return nil
}

func (gi *githubImporter) ensureCommentEdit(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, ghTargetId githubv4.ID, edit *userContentEdit) error {
	// find comment
	target, err := b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(ghTargetId))
	if err != nil {
		return err
	}
	// check if the comment edition already exist
	_, err = b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(edit.Id))
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

	if edit.DeletedAt != nil {
		// comment deletion, not supported yet
		return nil
	}

	commentId := entity.CombineIds(b.Id(), target)

	// comment edition
	_, err = b.EditCommentRaw(
		editor,
		edit.CreatedAt.Unix(),
		commentId,
		text.Cleanup(string(*edit.Diff)),
		map[string]string{
			metaKeyGithubId: parseId(edit.Id),
		},
	)

	if err != nil {
		return err
	}

	gi.out <- core.NewImportCommentEdition(b.Id(), commentId)
	return nil
}

func (gi *githubImporter) ensureComment(ctx context.Context, repo *cache.RepoCache, b *cache.BugCache, comment *issueComment, firstEdit *userContentEdit) error {
	author, err := gi.ensurePerson(ctx, repo, comment.Author)
	if err != nil {
		return err
	}

	_, err = b.ResolveOperationWithMetadata(metaKeyGithubId, parseId(comment.Id))
	if err == nil {
		return nil
	}
	if err != cache.ErrNoMatchingOp {
		// real error
		return err
	}

	var textInput string
	if firstEdit != nil {
		// use the first comment edit: it represents the comment creation itself
		textInput = string(*firstEdit.Diff)
	} else {
		// if there are not comment edits, then the comment struct holds the comment creation
		textInput = string(comment.Body)
	}

	// add comment operation
	commentId, _, err := b.AddCommentRaw(
		author,
		comment.CreatedAt.Unix(),
		text.Cleanup(textInput),
		nil,
		map[string]string{
			metaKeyGithubId:  parseId(comment.Id),
			metaKeyGithubUrl: comment.Url.String(),
		},
	)
	if err != nil {
		return err
	}

	gi.out <- core.NewImportComment(b.Id(), commentId)
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
	i, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyGithubLogin, string(actor.Login))
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

	i, err = repo.Identities().NewRaw(
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

func (gi *githubImporter) getGhost(ctx context.Context, repo *cache.RepoCache) (*cache.IdentityCache, error) {
	loginName := "ghost"
	// Look first in the cache
	i, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyGithubLogin, loginName)
	if err == nil {
		return i, nil
	}
	if entity.IsErrMultipleMatch(err) {
		return nil, err
	}
	user, err := gi.mediator.User(ctx, loginName)
	if err != nil {
		return nil, err
	}
	userName := ""
	if user.Name != nil {
		userName = string(*user.Name)
	}
	return repo.Identities().NewRaw(
		userName,
		"",
		string(user.Login),
		string(user.AvatarUrl),
		nil,
		map[string]string{
			metaKeyGithubLogin: string(user.Login),
		},
	)
}

// parseId converts the unusable githubv4.ID (an interface{}) into a string
func parseId(id githubv4.ID) string {
	return fmt.Sprintf("%v", id)
}
