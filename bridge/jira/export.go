package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/MichaelMure/git-bug/bridge/core"
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/cache"
	"github.com/MichaelMure/git-bug/entity"
)

var errDuplicateMatch = errors.New("Ambiguous match")

// jiraExporter implement the Exporter interface
type jiraExporter struct {
	conf core.Configuration

	// cache identities clients
	identityClient map[entity.Id]*Client

	// cache identifiers used to speed up exporting operations
	// cleared for each bug
	cachedOperationIDs map[entity.Id]string

	// cache labels used to speed up exporting labels events
	cachedLabels map[string]string

	// store JIRA project information
	project *Project
}

// Init .
func (self *jiraExporter) Init(conf core.Configuration) error {
	self.conf = conf
	//TODO: initialize with multiple tokens
	self.identityClient = make(map[entity.Id]*Client)
	self.cachedOperationIDs = make(map[entity.Id]string)
	self.cachedLabels = make(map[string]string)
	return nil
}

// getIdentityClient return an API client configured with the credentials
// of the given identity. If no client were found it will initialize it from
// the known credentials map and cache it for next use
func (self *jiraExporter) getIdentityClient(
	ctx *context.Context, id entity.Id) (*Client, error) {
	client, ok := self.identityClient[id]
	if ok {
		return client, nil
	}

	// TODO(josh)[]: The github exporter appears to contain code that will
	// allow it to export bugs owned by other people as long as we have a token
	// for that identity. I guess the equivalent for us will be as long as we
	// have a credentials pair for that identity.
	return nil, fmt.Errorf("Not implemented")
}

// ExportAll export all event made by the current user to Jira
func (self *jiraExporter) ExportAll(
	ctx context.Context, repo *cache.RepoCache, since time.Time) (
	<-chan core.ExportResult, error) {

	out := make(chan core.ExportResult)

	user, err := repo.GetUserIdentity()
	if err != nil {
		return nil, err
	}

	// TODO(josh)[]: The github exporter appears to contain code that will
	// allow it to export bugs owned by other people as long as we have a token
	// for that identity. I guess the equivalent for us will be as long as we
	// have a credentials pair for that identity.
	client := NewClient(self.conf[keyServer], &ctx)
	err = client.Login(self.conf)
	self.identityClient[user.Id()] = client

	if err != nil {
		return nil, err
	}

	client, err = self.getIdentityClient(&ctx, user.Id())
	if err != nil {
		return nil, err
	}

	self.project, err = client.GetProject(self.conf[keyProject])
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(out)

		var allIdentitiesIds []entity.Id
		for id := range self.identityClient {
			allIdentitiesIds = append(allIdentitiesIds, id)
		}

		allBugsIds := repo.AllBugsIds()

		for _, id := range allBugsIds {
			b, err := repo.ResolveBug(id)
			if err != nil {
				out <- core.NewExportError(errors.Wrap(err, "can't load bug"), id)
				return
			}

			select {

			case <-ctx.Done():
				// stop iterating if context cancel function is called
				return

			default:
				snapshot := b.Snapshot()

				// ignore issues whose last modification date is before the query date
				// TODO: compare the Lamport time instead of using the unix time
				if snapshot.CreatedAt.Before(since) {
					out <- core.NewExportNothing(b.Id(), "bug created before the since date")
					continue
				}

				if snapshot.HasAnyActor(allIdentitiesIds...) {
					// try to export the bug and it associated events
					self.exportBug(ctx, b, since, out)
				} else {
					out <- core.NewExportNothing(id, "not an actor")
				}
			}
		}
	}()

	return out, nil
}

// exportBug publish bugs and related events
func (self *jiraExporter) exportBug(
	ctx context.Context, b *cache.BugCache, since time.Time,
	out chan<- core.ExportResult) {
	snapshot := b.Snapshot()

	var bugJiraID string

	// Special case:
	// if a user try to export a bug that is not already exported to jira (or
	// imported from jira) and we do not have the token of the bug author,
	// there is nothing we can do.

	// first operation is always createOp
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	author := snapshot.Author

	// skip bug if it was imported from some other bug system
	origin, ok := snapshot.GetCreateMetadata(keyOrigin)
	if ok && origin != target {
		out <- core.NewExportNothing(
			b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
		return
	}

	// skip bug if it is a jira bug but is associated with another project
	// (one bridge per JIRA project)
	project, ok := snapshot.GetCreateMetadata(keyJiraProject)
	if ok && !stringInSlice(project, []string{self.project.ID, self.project.Key}) {
		out <- core.NewExportNothing(
			b.Id(), fmt.Sprintf("issue tagged with project: %s", project))
		return
	}

	// get jira bug ID
	jiraID, ok := snapshot.GetCreateMetadata(keyJiraID)
	if ok {
		out <- core.NewExportNothing(b.Id(), "bug creation already exported")
		// will be used to mark operation related to a bug as exported
		bugJiraID = jiraID
	} else {
		// check that we have credentials for operation author
		client, err := self.getIdentityClient(&ctx, author.Id())
		if err != nil {
			// if bug is not yet exported and we do not have the author's credentials
			// then there is nothing we can do, so just skip this bug
			out <- core.NewExportNothing(
				b.Id(), fmt.Sprintf("missing author token for user %.8s",
					author.Id().String()))
			return
		}

		// Load any custom fields required to create an issue from the git
		// config file.
		fields := make(map[string]interface{})
		defaultFields, hasConf := self.conf[keyCreateDefaults]
		if hasConf {
			json.Unmarshal([]byte(defaultFields), &fields)
		} else {
			// If there is no configuration provided, at the very least the
			// "issueType" field is always required. 10001 is "story" which I'm
			// pretty sure is standard/default on all JIRA instances.
			fields["issueType"] = "10001"
		}
		bugIDField, hasConf := self.conf[keyCreateGitBug]
		if hasConf {
			// If the git configuration also indicates it, we can assign the git-bug
			// id to a custom field to assist in integrations
			fields[bugIDField] = b.Id().String()
		}

		// create bug
		result, err := client.CreateIssue(
			self.project.ID, createOp.Title, createOp.Message, fields)
		if err != nil {
			err := errors.Wrap(err, "exporting jira issue")
			out <- core.NewExportError(err, b.Id())
			return
		}

		id := result.ID
		out <- core.NewExportBug(b.Id())
		// mark bug creation operation as exported
		err = markOperationAsExported(
			b, createOp.Id(), id, self.project.Key, time.Time{})
		if err != nil {
			err := errors.Wrap(err, "marking operation as exported")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// commit operation to avoid creating multiple issues with multiple pushes
		err = b.CommitAsNeeded()
		if err != nil {
			err := errors.Wrap(err, "bug commit")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// cache bug jira ID
		bugJiraID = id
	}

	// cache operation jira id
	self.cachedOperationIDs[createOp.Id()] = bugJiraID

	// lookup the mapping from git-bug "status" to JIRA "status" id
	statusMap := getStatusMap(self.conf)

	for _, op := range snapshot.Operations[1:] {
		// ignore SetMetadata operations
		if _, ok := op.(*bug.SetMetadataOperation); ok {
			continue
		}

		// ignore operations already existing in jira (due to import or export)
		// cache the ID of already exported or imported issues and events from
		// Jira
		if id, ok := op.GetMetadata(keyJiraID); ok {
			self.cachedOperationIDs[op.Id()] = id
			out <- core.NewExportNothing(op.Id(), "already exported operation")
			continue
		}

		opAuthor := op.GetAuthor()
		client, err := self.getIdentityClient(&ctx, opAuthor.Id())
		if err != nil {
			out <- core.NewExportNothing(
				op.Id(), fmt.Sprintf(
					"missing operation author credentials for user %.8s",
					author.Id().String()))
			continue
		}

		var id string
		var exportTime time.Time
		switch op.(type) {
		case *bug.AddCommentOperation:
			opr := op.(*bug.AddCommentOperation)
			comment, err := client.AddComment(bugJiraID, opr.Message)
			if err != nil {
				err := errors.Wrap(err, "adding comment")
				out <- core.NewExportError(err, b.Id())
				return
			}
			id = comment.ID
			out <- core.NewExportComment(op.Id())

			// cache comment id
			self.cachedOperationIDs[op.Id()] = id

		case *bug.EditCommentOperation:
			opr := op.(*bug.EditCommentOperation)
			if opr.Target == createOp.Id() {
				// An EditCommentOpreation with the Target set to the create operation
				// encodes a modification to the long-description/summary.
				exportTime, err = client.UpdateIssueBody(bugJiraID, opr.Message)
				if err != nil {
					err := errors.Wrap(err, "editing issue")
					out <- core.NewExportError(err, b.Id())
					return
				}
				out <- core.NewExportCommentEdition(op.Id())
				id = bugJiraID
			} else {
				// Otherwise it's an edit to an actual comment. A comment cannot be
				// edited before it was created, so it must be the case that we have
				// already observed and cached the AddCommentOperation.
				commentID, ok := self.cachedOperationIDs[opr.Target]
				if !ok {
					// Since an edit has to come after the creation, we expect we would
					// have cached the creation id.
					panic("unexpected error: comment id not found")
				}
				comment, err := client.UpdateComment(bugJiraID, commentID, opr.Message)
				if err != nil {
					err := errors.Wrap(err, "editing comment")
					out <- core.NewExportError(err, b.Id())
					return
				}
				out <- core.NewExportCommentEdition(op.Id())
				// JIRA doesn't track all comment edits, they will only tell us about
				// the most recent one. We must invent a consistent id for the operation
				// so we use the comment ID plus the timestamp of the update, as
				// reported by JIRA. Note that this must be consistent with the importer
				// during ensureComment()
				id = fmt.Sprintf("%s-%d", comment.ID, comment.Updated.Unix())
			}

		case *bug.SetStatusOperation:
			opr := op.(*bug.SetStatusOperation)
			jiraStatus, hasStatus := statusMap[opr.Status.String()]
			if hasStatus {
				exportTime, err = UpdateIssueStatus(client, bugJiraID, jiraStatus)
				if err != nil {
					err := errors.Wrap(err, "editing status")
					out <- core.NewExportError(err, b.Id())
					// Failure to update status isn't necessarily a big error. It's
					// possible that we just don't have enough information to make that
					// update. In this case, just don't export the operation.
					continue
				}
				out <- core.NewExportStatusChange(op.Id())
				// TODO(josh)[c2c6767]: query changelog to get the changelog-id so that
				// we don't re-import the same change.
				id = bugJiraID
			} else {
				out <- core.NewExportNothing(
					op.Id(), fmt.Sprintf(
						"No jira status mapped for %.8s", opr.Status.String()))
			}

		case *bug.SetTitleOperation:
			opr := op.(*bug.SetTitleOperation)
			exportTime, err = client.UpdateIssueTitle(bugJiraID, opr.Title)
			if err != nil {
				err := errors.Wrap(err, "editing title")
				out <- core.NewExportError(err, b.Id())
				return
			}
			out <- core.NewExportTitleEdition(op.Id())
			// TODO(josh)[c2c6767]: query changelog to get the changelog-id so that
			// we don't re-import the same change.
			id = bugJiraID

		case *bug.LabelChangeOperation:
			opr := op.(*bug.LabelChangeOperation)
			exportTime, err = client.UpdateLabels(
				bugJiraID, opr.Added, opr.Removed)
			if err != nil {
				err := errors.Wrap(err, "updating labels")
				out <- core.NewExportError(err, b.Id())
				return
			}
			out <- core.NewExportLabelChange(op.Id())
			// TODO(josh)[c2c6767]: query changelog to get the changelog-id so that
			// we don't re-import the same change.
			id = bugJiraID

		default:
			panic("unhandled operation type case")
		}

		// mark operation as exported
		// TODO(josh)[c2c6767]: Should we query the changelog after we export?
		// Some of the operations above don't record an ID... so we are bound to
		// re-import them. It shouldn't cause too much of an issue but we will have
		// duplicate edit entries for everything and it would be nice to avoid that.
		err = markOperationAsExported(
			b, op.Id(), id, self.project.Key, exportTime)
		if err != nil {
			err := errors.Wrap(err, "marking operation as exported")
			out <- core.NewExportError(err, b.Id())
			return
		}

		// commit at each operation export to avoid exporting same events multiple
		// times
		err = b.CommitAsNeeded()
		if err != nil {
			err := errors.Wrap(err, "bug commit")
			out <- core.NewExportError(err, b.Id())
			return
		}
	}
}

func markOperationAsExported(
	b *cache.BugCache, target entity.Id, jiraID, jiraProject string,
	exportTime time.Time) error {

	newMetadata := map[string]string{
		keyJiraID:      jiraID,
		keyJiraProject: jiraProject,
	}
	if !exportTime.IsZero() {
		newMetadata[keyJiraExportTime] = exportTime.Format(http.TimeFormat)
	}

	_, err := b.SetMetadata(target, newMetadata)
	return err
}

// UpdateIssueStatus attempts to change the "status" field by finding a
// transition which achieves the desired state and then performing that
// transition
func UpdateIssueStatus(
	client *Client, issueKeyOrID string, desiredStateNameOrID string) (
	time.Time, error) {

	var responseTime time.Time

	tlist, err := client.GetTransitions(issueKeyOrID)
	if err != nil {
		return responseTime, err
	}

	transition := getTransitionTo(tlist, desiredStateNameOrID)
	if transition == nil {
		return responseTime, errTransitionNotFound
	}

	responseTime, err = client.DoTransition(issueKeyOrID, transition.ID)
	if err != nil {
		return responseTime, err
	}

	return responseTime, nil
}
