package ado

import (
	"context"
	"fmt"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
)

// adoImporter implements the core.Importer interface.
type adoImporter struct {
	conf   core.Configuration
	client *Client

	// send-only channel
	out chan<- core.ImportResult
}

// Init loads the PAT credential and builds the REST client.
func (ai *adoImporter) Init(ctx context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	ai.conf = conf

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
		return fmt.Errorf("no credential for this bridge, run \"git bug bridge auth\" or reconfigure")
	}

	ai.client, err = buildClient(ctx, conf[confKeyBaseURL], conf[confKeyOrganization], conf[confKeyProject], creds[0])
	return err
}

// ImportAll imports all work items changed since the given time.
func (ai *adoImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	out := make(chan core.ImportResult)
	ai.out = out

	go func() {
		defer close(out)

		ids, err := ai.client.SearchWorkItemIDs(buildWiql(ai.conf[confKeyProject], since))
		if err != nil {
			out <- core.NewImportError(fmt.Errorf("wiql query: %w", err), "")
			return
		}

		items, err := ai.client.GetWorkItems(ids)
		if err != nil {
			out <- core.NewImportError(fmt.Errorf("fetching work items: %w", err), "")
			return
		}

		for _, wi := range items {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if !since.IsZero() && !wi.Fields.ChangedDate.IsZero() && wi.Fields.ChangedDate.Before(since) {
				continue
			}

			b, err := ai.ensureWorkItem(repo, wi)
			if err != nil {
				out <- core.NewImportError(fmt.Errorf("work item %d: %w", wi.ID, err), "")
				return
			}

			comments, err := ai.client.GetComments(wi.ID)
			if err != nil {
				out <- core.NewImportError(fmt.Errorf("comments for %d: %w", wi.ID, err), b.Id())
				return
			}
			for _, cm := range comments {
				if err := ai.ensureComment(repo, b, cm); err != nil {
					out <- core.NewImportError(err, b.Id())
				}
			}

			updates, err := ai.client.GetUpdates(wi.ID)
			if err != nil {
				out <- core.NewImportError(fmt.Errorf("updates for %d: %w", wi.ID, err), b.Id())
				return
			}
			if err := ai.ensureUpdates(repo, b, wi, updates); err != nil {
				out <- core.NewImportError(err, b.Id())
			}

			if !b.NeedCommit() {
				out <- core.NewImportNothing(b.Id(), "no imported operation")
			} else if err := b.Commit(); err != nil {
				out <- core.NewImportError(fmt.Errorf("bug commit: %w", err), b.Id())
				return
			}
		}
	}()

	return out, nil
}

// ensurePerson creates or retrieves the identity for an Azure DevOps user.
func (ai *adoImporter) ensurePerson(repo *cache.RepoCache, user Identity) (*cache.IdentityCache, error) {
	key := user.Key()
	if key == "" {
		key = "unknown"
	}

	i, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyAdoUser, key)
	if err == nil {
		return i, nil
	}
	if _, ok := err.(entity.ErrMultipleMatch); ok {
		return nil, err
	}

	name := user.DisplayName
	if name == "" {
		name = key
	}

	i, err = repo.Identities().NewRaw(
		name,
		user.Email(),
		key,
		"",
		nil,
		map[string]string{
			metaKeyAdoUser: key,
		},
	)
	if err != nil {
		return nil, err
	}

	ai.out <- core.NewImportIdentity(i.Id())
	return i, nil
}

// ensureWorkItem creates a git-bug bug for an Azure DevOps work item.
func (ai *adoImporter) ensureWorkItem(repo *cache.RepoCache, wi WorkItem) (*cache.BugCache, error) {
	author, err := ai.ensurePerson(repo, wi.Fields.CreatedBy)
	if err != nil {
		return nil, err
	}

	adoID := fmt.Sprintf("%d", wi.ID)

	b, err := repo.Bugs().ResolveMatcher(func(excerpt *cache.BugExcerpt) bool {
		if base, ok := excerpt.CreateMetadata[metaKeyAdoBaseURL]; ok && base != ai.conf[confKeyBaseURL] {
			return false
		}
		return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
			excerpt.CreateMetadata[metaKeyAdoID] == adoID &&
			excerpt.CreateMetadata[metaKeyAdoProject] == ai.conf[confKeyProject]
	})
	if err != nil && !entity.IsErrNotFound(err) {
		return nil, err
	}

	if entity.IsErrNotFound(err) {
		createdAt := wi.Fields.CreatedDate
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		b, _, err = repo.Bugs().NewRaw(
			author,
			createdAt.Unix(),
			text.CleanupOneLine(wi.Fields.Title),
			text.Cleanup(wi.Fields.Description),
			nil,
			map[string]string{
				core.MetaKeyOrigin: target,
				metaKeyAdoID:       adoID,
				metaKeyAdoURL:      wi.URL,
				metaKeyAdoProject:  ai.conf[confKeyProject],
				metaKeyAdoOrg:      ai.conf[confKeyOrganization],
				metaKeyAdoBaseURL:  ai.conf[confKeyBaseURL],
			},
		)
		if err != nil {
			return nil, err
		}
		ai.out <- core.NewImportBug(b.Id())
	}

	return b, nil
}

// ensureComment imports a single work item comment.
func (ai *adoImporter) ensureComment(repo *cache.RepoCache, b *cache.BugCache, cm Comment) error {
	author, err := ai.ensurePerson(repo, cm.CreatedBy)
	if err != nil {
		return err
	}

	commentAdoID := fmt.Sprintf("comment-%d", cm.ID)

	_, err = b.ResolveOperationWithMetadata(metaKeyAdoID, commentAdoID)
	if err != nil && err != cache.ErrNoMatchingOp {
		return err
	}
	if err == nil {
		// already imported
		return nil
	}

	createdAt := cm.CreatedDate
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	commentID, _, err := b.AddCommentRaw(
		author,
		createdAt.Unix(),
		text.Cleanup(cm.Text),
		nil,
		map[string]string{
			metaKeyAdoID: commentAdoID,
		},
	)
	if err != nil {
		return err
	}

	ai.out <- core.NewImportComment(b.Id(), commentID)
	return nil
}

// ensureUpdates replays the revision history of a work item into git-bug
// operations for status, title, tags and description changes.
func (ai *adoImporter) ensureUpdates(repo *cache.RepoCache, b *cache.BugCache, wi WorkItem, updates []WorkItemUpdate) error {
	closedStates := closedStatesFromConf(ai.conf)

	// Map exported operations by their export time so we can recognise the
	// Azure DevOps revisions that we produced ourselves and avoid re-importing
	// them.
	exportedByTime := make(map[int64]entity.Id)
	for _, op := range b.Snapshot().Operations {
		if ts, ok := op.GetMetadata(metaKeyAdoExportTime); ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				exportedByTime[t.Unix()] = op.Id()
			}
		}
	}

	for _, up := range updates {
		// rev 1 is the creation, already handled by ensureWorkItem.
		if up.Rev <= 1 {
			continue
		}

		// Skip revisions we exported ourselves, tagging the causing operation.
		if opID, ok := exportedByTime[up.RevisedDate.Unix()]; ok {
			derivedID := fmt.Sprintf("%d-%d", wi.ID, up.Rev)
			if _, err := b.SetMetadata(opID, map[string]string{metaKeyAdoDerivedID: derivedID}); err != nil {
				return err
			}
			continue
		}

		author, err := ai.ensurePerson(repo, up.RevisedBy)
		if err != nil {
			return err
		}
		revisedAt := up.RevisedDate
		if revisedAt.IsZero() {
			revisedAt = wi.Fields.ChangedDate
		}

		if err := ai.importStateChange(b, author, wi, up, revisedAt.Unix(), closedStates); err != nil {
			return err
		}
		if err := ai.importTitleChange(b, author, wi, up, revisedAt.Unix()); err != nil {
			return err
		}
		if err := ai.importTagsChange(b, author, wi, up, revisedAt.Unix()); err != nil {
			return err
		}
		if err := ai.importDescriptionChange(b, author, wi, up, revisedAt.Unix()); err != nil {
			return err
		}
	}

	return ai.reconcileStatus(repo, b, wi, closedStates)
}

// reconcileStatus corrects the bug's final status to match the work item's
// current state. It covers work items created directly in a closed state, whose
// creation revision (skipped above) carries no state transition to replay.
func (ai *adoImporter) reconcileStatus(repo *cache.RepoCache, b *cache.BugCache, wi WorkItem, closedStates []string) error {
	desiredClosed := isClosedState(wi.Fields.State, closedStates)
	if desiredClosed == (b.Snapshot().Status == common.ClosedStatus) {
		return nil
	}

	author, err := ai.ensurePerson(repo, wi.Fields.CreatedBy)
	if err != nil {
		return err
	}
	when := wi.Fields.ChangedDate
	if when.IsZero() {
		when = time.Now()
	}

	metadata := map[string]string{
		metaKeyAdoID:        fmt.Sprintf("%d", wi.ID),
		metaKeyAdoDerivedID: fmt.Sprintf("%d-currentstate", wi.ID),
	}

	var op interface{ Id() entity.Id }
	if desiredClosed {
		op, err = b.CloseRaw(author, when.Unix(), metadata)
	} else {
		op, err = b.OpenRaw(author, when.Unix(), metadata)
	}
	if err != nil {
		return err
	}
	ai.out <- core.NewImportStatusChange(b.Id(), op.Id())
	return nil
}

func (ai *adoImporter) alreadyImported(b *cache.BugCache, derivedID string) (bool, error) {
	_, err := b.ResolveOperationWithMetadata(metaKeyAdoDerivedID, derivedID)
	if err == nil {
		return true, nil
	}
	if err != cache.ErrNoMatchingOp {
		return false, err
	}
	return false, nil
}

func (ai *adoImporter) importStateChange(b *cache.BugCache, author *cache.IdentityCache, wi WorkItem, up WorkItemUpdate, unixTime int64, closedStates []string) error {
	fu, ok := up.Fields["System.State"]
	if !ok {
		return nil
	}
	oldClosed := isClosedState(fieldString(fu.OldValue), closedStates)
	newClosed := isClosedState(fieldString(fu.NewValue), closedStates)
	if oldClosed == newClosed {
		// open->open or closed->closed: no git-bug status change
		return nil
	}

	derivedID := fmt.Sprintf("%d-%d-state", wi.ID, up.Rev)
	done, err := ai.alreadyImported(b, derivedID)
	if err != nil || done {
		return err
	}

	metadata := map[string]string{
		metaKeyAdoID:        fmt.Sprintf("%d", wi.ID),
		metaKeyAdoDerivedID: derivedID,
	}

	var op interface{ Id() entity.Id }
	if newClosed {
		op, err = b.CloseRaw(author, unixTime, metadata)
	} else {
		op, err = b.OpenRaw(author, unixTime, metadata)
	}
	if err != nil {
		return err
	}
	ai.out <- core.NewImportStatusChange(b.Id(), op.Id())
	return nil
}

func (ai *adoImporter) importTitleChange(b *cache.BugCache, author *cache.IdentityCache, wi WorkItem, up WorkItemUpdate, unixTime int64) error {
	fu, ok := up.Fields["System.Title"]
	if !ok {
		return nil
	}
	newTitle := fieldString(fu.NewValue)

	derivedID := fmt.Sprintf("%d-%d-title", wi.ID, up.Rev)
	done, err := ai.alreadyImported(b, derivedID)
	if err != nil || done {
		return err
	}

	op, err := b.SetTitleRaw(author, unixTime, text.CleanupOneLine(newTitle), map[string]string{
		metaKeyAdoID:        fmt.Sprintf("%d", wi.ID),
		metaKeyAdoDerivedID: derivedID,
	})
	if err != nil {
		return err
	}
	ai.out <- core.NewImportTitleEdition(b.Id(), op.Id())
	return nil
}

func (ai *adoImporter) importTagsChange(b *cache.BugCache, author *cache.IdentityCache, wi WorkItem, up WorkItemUpdate, unixTime int64) error {
	fu, ok := up.Fields["System.Tags"]
	if !ok {
		return nil
	}
	added, removed := tagSetDifference(parseTags(fieldString(fu.OldValue)), parseTags(fieldString(fu.NewValue)))
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}

	derivedID := fmt.Sprintf("%d-%d-tags", wi.ID, up.Rev)
	done, err := ai.alreadyImported(b, derivedID)
	if err != nil || done {
		return err
	}

	op, err := b.ForceChangeLabelsRaw(
		author,
		unixTime,
		text.CleanupOneLineArray(added),
		text.CleanupOneLineArray(removed),
		map[string]string{
			metaKeyAdoID:        fmt.Sprintf("%d", wi.ID),
			metaKeyAdoDerivedID: derivedID,
		},
	)
	if err != nil {
		return err
	}
	ai.out <- core.NewImportLabelChange(b.Id(), op.Id())
	return nil
}

func (ai *adoImporter) importDescriptionChange(b *cache.BugCache, author *cache.IdentityCache, wi WorkItem, up WorkItemUpdate, unixTime int64) error {
	fu, ok := up.Fields["System.Description"]
	if !ok {
		return nil
	}
	newBody := fieldString(fu.NewValue)

	derivedID := fmt.Sprintf("%d-%d-description", wi.ID, up.Rev)
	done, err := ai.alreadyImported(b, derivedID)
	if err != nil || done {
		return err
	}

	commentID, _, err := b.EditCreateCommentRaw(
		author,
		unixTime,
		text.Cleanup(newBody),
		map[string]string{
			metaKeyAdoID:        fmt.Sprintf("%d", wi.ID),
			metaKeyAdoDerivedID: derivedID,
		},
	)
	if err != nil {
		return err
	}
	ai.out <- core.NewImportCommentEdition(b.Id(), commentID)
	return nil
}
