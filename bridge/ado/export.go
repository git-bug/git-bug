package ado

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

// adoExporter implements the core.Exporter interface.
type adoExporter struct {
	conf core.Configuration

	// one client per known identity, plus a default used when a bug's author
	// has no dedicated credential (the common case for a single service PAT).
	identityClient map[entity.Id]*Client
	defaultClient  *Client

	// git-bug operation id -> Azure DevOps id (work item id, or numeric comment
	// id for AddComment operations). Cleared implicitly per run.
	cachedOperationIDs map[entity.Id]string

	project *Project
}

// Init loads every PAT credential of the bridge and prepares one client each.
func (ae *adoExporter) Init(ctx context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	ae.conf = conf
	ae.identityClient = make(map[entity.Id]*Client)
	ae.cachedOperationIDs = make(map[entity.Id]string)

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyBaseURL]),
	)
	if err != nil {
		return err
	}

	for _, cred := range creds {
		login, ok := cred.GetMetadata(auth.MetaKeyLogin)
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "credential %s is not tagged with an Azure DevOps login\n", cred.ID().Human())
			continue
		}

		client, err := buildClient(ctx, conf[confKeyBaseURL], conf[confKeyOrganization], conf[confKeyProject], cred)
		if err != nil {
			return err
		}
		if ae.defaultClient == nil {
			ae.defaultClient = client
		}

		user, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyAdoLogin, login)
		if entity.IsErrNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if _, ok := ae.identityClient[user.Id()]; !ok {
			ae.identityClient[user.Id()] = client
		}
	}

	if ae.defaultClient == nil {
		return fmt.Errorf("no credentials for this bridge")
	}

	ae.project, err = ae.defaultClient.GetProject()
	return err
}

// getClientForIdentity returns the client for a given identity, falling back to
// the default client so a single service PAT can export every bug.
func (ae *adoExporter) getClientForIdentity(userID entity.Id) *Client {
	if client, ok := ae.identityClient[userID]; ok {
		return client
	}
	return ae.defaultClient
}

// ExportAll exports all local bugs and their operations to Azure DevOps.
func (ae *adoExporter) ExportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	out := make(chan core.ExportResult)

	go func() {
		defer close(out)

		for _, id := range repo.Bugs().AllIds() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			b, err := repo.Bugs().Resolve(id)
			if err != nil {
				out <- core.NewExportError(fmt.Errorf("can't load bug: %w", err), id)
				return
			}

			snapshot := b.Snapshot()
			if snapshot.CreateTime.Before(since) {
				out <- core.NewExportNothing(id, "bug created before the since date")
				continue
			}

			if err := ae.exportBug(b, out); err != nil {
				out <- core.NewExportError(fmt.Errorf("can't export bug: %w", err), id)
				return
			}
		}
	}()

	return out, nil
}

// exportBug publishes a bug and its operations to Azure DevOps.
func (ae *adoExporter) exportBug(b *cache.BugCache, out chan<- core.ExportResult) error {
	snapshot := b.Snapshot()
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	author := snapshot.Author

	// skip bugs imported from another tracker
	if origin, ok := snapshot.GetCreateMetadata(core.MetaKeyOrigin); ok && origin != target {
		out <- core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
		return nil
	}

	// skip bugs belonging to another Azure DevOps project (one bridge per project)
	if project, ok := snapshot.GetCreateMetadata(metaKeyAdoProject); ok &&
		project != ae.project.ID && project != ae.project.Name {
		out <- core.NewExportNothing(b.Id(), fmt.Sprintf("issue tagged with project: %s", project))
		return nil
	}

	var bugAdoID string
	if id, ok := snapshot.GetCreateMetadata(metaKeyAdoID); ok {
		bugAdoID = id
	} else {
		client := ae.getClientForIdentity(author.Id())

		ops := []patchOp{
			{Op: "add", Path: "/fields/System.Title", Value: createOp.Title},
		}
		if createOp.Message != "" {
			ops = append(ops, patchOp{Op: "add", Path: "/fields/System.Description", Value: createOp.Message})
		}

		workItemType := confOr(ae.conf, confKeyCreateType, defaultCreateType)
		result, err := client.CreateWorkItem(workItemType, ops)
		if err != nil {
			out <- core.NewExportError(fmt.Errorf("creating work item: %w", err), b.Id())
			return err
		}

		bugAdoID = strconv.Itoa(result.ID)
		out <- core.NewExportBug(b.Id())

		if err := ae.markExported(b, createOp.Id(), bugAdoID, result.URL, time.Time{}); err != nil {
			out <- core.NewExportError(fmt.Errorf("marking operation as exported: %w", err), b.Id())
			return err
		}
		if err := b.CommitAsNeeded(); err != nil {
			out <- core.NewExportError(fmt.Errorf("bug commit: %w", err), b.Id())
			return err
		}
	}

	ae.cachedOperationIDs[createOp.Id()] = bugAdoID

	adoIDInt, err := strconv.Atoi(bugAdoID)
	if err != nil {
		return fmt.Errorf("invalid Azure DevOps id %q: %w", bugAdoID, err)
	}

	for _, op := range snapshot.Operations[1:] {
		if _, ok := op.(dag.OperationDoesntChangeSnapshot); ok {
			continue
		}
		// skip operations already present in Azure DevOps (imported or exported)
		if id, ok := op.GetMetadata(metaKeyAdoID); ok {
			ae.cachedOperationIDs[op.Id()] = id
			continue
		}

		client := ae.getClientForIdentity(op.Author().Id())

		var exportedID string
		var exportTime time.Time

		switch opr := op.(type) {
		case *bug.AddCommentOperation:
			comment, err := client.AddComment(adoIDInt, opr.Message)
			if err != nil {
				out <- core.NewExportError(fmt.Errorf("adding comment: %w", err), b.Id())
				return err
			}
			// match the importer's comment id format for idempotency
			exportedID = fmt.Sprintf("comment-%d", comment.ID)
			ae.cachedOperationIDs[op.Id()] = strconv.Itoa(comment.ID)
			out <- core.NewExportComment(b.Id())

		case *bug.EditCommentOperation:
			if opr.Target == createOp.Id() {
				wi, err := client.UpdateWorkItem(adoIDInt, []patchOp{
					{Op: "add", Path: "/fields/System.Description", Value: opr.Message},
				})
				if err != nil {
					out <- core.NewExportError(fmt.Errorf("editing description: %w", err), b.Id())
					return err
				}
				exportedID = bugAdoID
				exportTime = wi.Fields.ChangedDate
				out <- core.NewExportCommentEdition(b.Id())
			} else {
				commentIDStr, ok := ae.cachedOperationIDs[opr.Target]
				if !ok {
					return fmt.Errorf("comment id not found for edit operation")
				}
				commentID, err := strconv.Atoi(commentIDStr)
				if err != nil {
					return fmt.Errorf("invalid comment id %q: %w", commentIDStr, err)
				}
				comment, err := client.UpdateComment(adoIDInt, commentID, opr.Message)
				if err != nil {
					out <- core.NewExportError(fmt.Errorf("editing comment: %w", err), b.Id())
					return err
				}
				exportedID = fmt.Sprintf("comment-%d", comment.ID)
				out <- core.NewExportCommentEdition(b.Id())
			}

		case *bug.SetStatusOperation:
			wi, err := client.UpdateWorkItem(adoIDInt, []patchOp{
				{Op: "add", Path: "/fields/System.State", Value: ae.targetState(opr.Status)},
			})
			if err != nil {
				// a failed transition is not fatal: the target state may not exist
				// in this project's process.
				out <- core.NewExportWarning(fmt.Errorf("editing state: %w", err), b.Id())
				continue
			}
			exportedID = bugAdoID
			exportTime = wi.Fields.ChangedDate
			out <- core.NewExportStatusChange(b.Id())

		case *bug.SetTitleOperation:
			wi, err := client.UpdateWorkItem(adoIDInt, []patchOp{
				{Op: "add", Path: "/fields/System.Title", Value: opr.Title},
			})
			if err != nil {
				out <- core.NewExportError(fmt.Errorf("editing title: %w", err), b.Id())
				return err
			}
			exportedID = bugAdoID
			exportTime = wi.Fields.ChangedDate
			out <- core.NewExportTitleEdition(b.Id())

		case *bug.LabelChangeOperation:
			wi, err := ae.exportLabelChange(client, adoIDInt, opr)
			if err != nil {
				out <- core.NewExportError(fmt.Errorf("updating tags: %w", err), b.Id())
				return err
			}
			exportedID = bugAdoID
			exportTime = wi.Fields.ChangedDate
			out <- core.NewExportLabelChange(b.Id())

		default:
			continue
		}

		if err := ae.markExported(b, op.Id(), exportedID, "", exportTime); err != nil {
			out <- core.NewExportError(fmt.Errorf("marking operation as exported: %w", err), b.Id())
			return err
		}
		if err := b.CommitAsNeeded(); err != nil {
			out <- core.NewExportError(fmt.Errorf("bug commit: %w", err), b.Id())
			return err
		}
	}

	return nil
}

// exportLabelChange applies a git-bug label change to the work item's tags by
// reading the current tags, applying the delta and writing them back.
func (ae *adoExporter) exportLabelChange(client *Client, adoID int, opr *bug.LabelChangeOperation) (*WorkItem, error) {
	wi, err := client.GetWorkItem(adoID)
	if err != nil {
		return nil, err
	}

	tags := parseTags(wi.Fields.Tags)
	present := make(map[string]bool, len(tags))
	for _, t := range tags {
		present[t] = true
	}
	for _, l := range opr.Added {
		if s := l.String(); !present[s] {
			tags = append(tags, s)
			present[s] = true
		}
	}
	removed := make(map[string]bool, len(opr.Removed))
	for _, l := range opr.Removed {
		removed[l.String()] = true
	}
	kept := make([]string, 0, len(tags))
	for _, t := range tags {
		if !removed[t] {
			kept = append(kept, t)
		}
	}

	return client.UpdateWorkItem(adoID, []patchOp{
		{Op: "add", Path: "/fields/System.Tags", Value: joinTags(kept)},
	})
}

// targetState maps a git-bug status to the configured Azure DevOps state name.
func (ae *adoExporter) targetState(status common.Status) string {
	if status == common.ClosedStatus {
		return confOr(ae.conf, confKeyClosedStateTarget, defaultClosedStateTarget)
	}
	return confOr(ae.conf, confKeyOpenStateTarget, defaultOpenStateTarget)
}

// markExported tags a git-bug operation with the Azure DevOps id it maps to.
func (ae *adoExporter) markExported(b *cache.BugCache, opID entity.Id, adoID, url string, exportTime time.Time) error {
	metadata := map[string]string{
		metaKeyAdoID:      adoID,
		metaKeyAdoProject: ae.conf[confKeyProject],
	}
	if url != "" {
		metadata[metaKeyAdoURL] = url
	}
	if !exportTime.IsZero() {
		metadata[metaKeyAdoExportTime] = exportTime.UTC().Format(time.RFC3339)
	}
	_, err := b.SetMetadata(opID, metadata)
	return err
}

// confOr returns conf[key] if set and non-empty, else the default.
func confOr(conf core.Configuration, key, def string) string {
	if v, ok := conf[key]; ok && v != "" {
		return v
	}
	return def
}
