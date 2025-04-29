package todosrht

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/entity/dag"
)

var (
	ErrMissingCredentials = errors.New("missing credentials")
)

// todosrhtExporter implement the Exporter interface
type todosrhtExporter struct {
	conf core.Configuration

	// cache identities clients
	identityClient map[entity.Id]TodosrhtClient

	// cache identifiers used to speed up exporting operations
	// cleared for each bug
	cachedOperationIDs map[entity.Id]string

	// cache labels used to speed up exporting labels events
	cachedLabels map[string]string

	// store TODOSRHT tracker information
	tracker *Tracker
}

// Init .
func (je *todosrhtExporter) Init(ctx context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	je.conf = conf
	je.identityClient = make(map[entity.Id]TodosrhtClient)
	je.cachedOperationIDs = make(map[entity.Id]string)

	// Preload all clients based on configured tokens
	err := je.cacheAllClient(ctx, repo)
	if err != nil {
		return err
	}

	if len(je.identityClient) == 0 {
		return fmt.Errorf("no credentials for this bridge")
	}

	// Use the client associated with the default login for general tracker operations
	defaultLogin := je.conf[confKeyDefaultLogin]
	var defaultClient *TodoSClient
	for _, client := range je.identityClient {
		// We don't have a direct way to map a client back to its login here without iterating metadata again
		// For now, assume the first client in the map is sufficient for initial tracker lookup
		// A more robust solution might involve storing the default client by ID or login in the exporter struct.
		defaultClient = client.(*TodoSClient)
		break
	}
	if defaultClient == nil {
		return fmt.Errorf("could not find a client for the default login: %s", defaultLogin)
	}

	// Get tracker information

	trackerName := je.conf[confKeyTrackerName]
	tracker, err := defaultClient.GetTracker(ctx, trackerName)
	if err != nil {
		return fmt.Errorf("failed to get tracker '%s': %w", trackerName, err)
	}
	je.tracker = tracker

	return nil
}

func (je *todosrhtExporter) cacheAllClient(ctx context.Context, repo *cache.RepoCache) error {
	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyBaseURL, je.conf[confKeyBaseUrl]),
	)
	if err != nil {
		return err
	}

	for _, cred := range creds {
		login, ok := cred.GetMetadata(auth.MetaKeyLogin)
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "credential %s is not tagged with a SourceHut login\n", cred.ID().Human())
			continue
		}

		user, err := repo.Identities().ResolveIdentityImmutableMetadata(metaKeyTodoSourceHutLogin, login)
		if entity.IsErrNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}

		if _, ok := je.identityClient[user.Id()]; !ok {
			tokenCred, ok := cred.(*auth.Token)
			if !ok {
				return fmt.Errorf("expected token credential, got %T", cred)
			}
			client := NewTodoSClient(ctx, je.conf[confKeyBaseUrl], tokenCred.Value)
			je.identityClient[user.Id()] = client
		}
	}

	return nil
}

// getClientForIdentity return an API client configured with the credentials
// of the given identity. If no client were found it will initialize it from
// the known credentials and cache it for next use.
func (je *todosrhtExporter) getClientForIdentity(userId entity.Id) (TodosrhtClient, error) {
	client, ok := je.identityClient[userId]
	if ok {
		return client, nil
	}

	return nil, ErrMissingCredentials
}

// ExportAll export all event made by the current user to TodoSourceHut
func (je *todosrhtExporter) ExportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ExportResult, error) {
	out := make(chan core.ExportResult)

	go func() {
		defer close(out)

		var allIdentitiesIds []entity.Id
		for id := range je.identityClient {
			allIdentitiesIds = append(allIdentitiesIds, id)
		}

		allBugsIds := repo.Bugs().AllIds()

		for _, id := range allBugsIds {
			b, err := repo.Bugs().Resolve(id)
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
				if snapshot.CreateTime.Before(since) {
					out <- core.NewExportNothing(b.Id(), "bug created before the since date")
					continue
				}

				if snapshot.HasAnyActor(allIdentitiesIds...) {
					// try to export the bug and it associated events
					err := je.exportBug(ctx, b, out)
					if err != nil {
						out <- core.NewExportError(errors.Wrap(err, "can't export bug"), id)
						return
					}
				} else {
					out <- core.NewExportNothing(id, "not an actor")
				}
			}
		}
	}()

	return out, nil
}

// exportBug publish bugs and related events
func (je *todosrhtExporter) exportBug(ctx context.Context, b *cache.BugCache, out chan<- core.ExportResult) error {
	snapshot := b.Snapshot()

	var bugTodoSourceHutID int // SourceHut ticket ID is an integer

	// first operation is always createOp
	createOp := snapshot.Operations[0].(*bug.CreateOperation)
	author := snapshot.Author

	// skip bug if it was imported from some other bug system
	origin, ok := snapshot.GetCreateMetadata(core.MetaKeyOrigin)
	if ok && origin != target {
		out <- core.NewExportNothing(
			b.Id(), fmt.Sprintf("issue tagged with origin: %s", origin))
		return nil
	}

	// skip bug if it is a todosrht bug but is associated with another tracker
	trackerMeta, ok := snapshot.GetCreateMetadata(metaKeyTodoSourceHutTracker)
	if ok && trackerMeta != je.conf[confKeyTrackerName] {
		out <- core.NewExportNothing(
			b.Id(), fmt.Sprintf("issue tagged with tracker: %s", trackerMeta))
		return nil
	}

	// get todosrht bug ID
	todosrhtIDStr, ok := snapshot.GetCreateMetadata(metaKeyTodoSourceHutId)
	if ok {
		var err error
		bugTodoSourceHutID, err = strconv.Atoi(todosrhtIDStr)
		if err != nil {
			return errors.Wrap(err, "invalid todosrht ID in metadata")
		}
	} else {
		// check that we have credentials for operation author
		client, err := je.getClientForIdentity(author.Id())
		if err != nil {
			// if bug is not yet exported and we do not have the author's credentials
			// then there is nothing to do, so just skip this bug
			out <- core.NewExportNothing(
				b.Id(), fmt.Sprintf("missing author credentials for user %.8s",
					author.Id().String()))
			return nil
		}

		// create bug
		ticket, err := createTodoSRHTTicket(ctx, client, je.tracker.Id, createOp.Title, createOp.Message)
		if err != nil {
			err := errors.Wrap(err, "exporting todosrht issue")
			out <- core.NewExportError(err, b.Id())
			return err
		}

		bugTodoSourceHutID = ticket.Id
		out <- core.NewExportBug(b.Id())
		// mark bug creation operation as exported
		err = markOperationAsExported(
			b, createOp.Id(), fmt.Sprintf("%d", ticket.Id), je.tracker.Name, ticket.Created.Unix())
		if err != nil {
			err := errors.Wrap(err, "marking operation as exported")
			out <- core.NewExportError(err, b.Id())
			return err
		}

		// commit operation to avoid creating multiple issues with multiple pushes
		err = b.CommitAsNeeded()
		if err != nil {
			err := errors.Wrap(err, "bug commit")
			out <- core.NewExportError(err, b.Id())
			return err
		}
	}

	// cache operation todosrht id
	je.cachedOperationIDs[createOp.Id()] = fmt.Sprintf("%d", bugTodoSourceHutID)

	for _, op := range snapshot.Operations[1:] {
		// ignore SetMetadata operations
		if _, ok := op.(dag.OperationDoesntChangeSnapshot); ok {
			continue
		}

		// ignore operations already existing in todosrht (due to import or export)
		// cache the ID of already exported or imported issues and events from
		// TodoSourceHut
		if id, ok := op.GetMetadata(metaKeyTodoSourceHutId); ok {
			je.cachedOperationIDs[op.Id()] = id
			continue
		}

		opAuthor := op.Author()
		client, err := je.getClientForIdentity(opAuthor.Id())
		if err != nil {
			out <- core.NewExportError(
				fmt.Errorf("missing operation author credentials for user %.8s",
					opAuthor.Id().String()), b.Id())
			continue
		}

		var todosrhtOpID string
		var exportTime int64 // SourceHut uses Unix timestamp for Created/Updated

		switch opr := op.(type) {
		case *bug.AddCommentOperation:
			commentEvent, err := addTodoSRHTComment(ctx, client, je.tracker.Id, bugTodoSourceHutID, opr.Message)
			if err != nil {
				err := errors.Wrap(err, "adding comment")
				out <- core.NewExportError(err, b.Id())
				return err
			}
			todosrhtOpID = fmt.Sprintf("%d", commentEvent.Id)
			exportTime = commentEvent.Created.Unix()
			out <- core.NewExportComment(b.Id())

			// cache comment id
			je.cachedOperationIDs[op.Id()] = todosrhtOpID

		case *bug.EditCommentOperation:
			// SourceHut doesn't have explicit "edit comment" mutations.
			// It tracks comments as events, and comments can be superseded by new ones.
			// For simplicity and to fit git-bug's model, we'll treat
			// EditCommentOperation targeting the initial ticket body as an UpdateTicket
			// and ignore actual comment edits for now if they are not the initial body.
			// A more complex implementation would involve trying to find the last comment
			// and creating a new one that supersedes it.
			if opr.Target == createOp.Id() {
				ticket, err := updateTodoSRHTTicketBody(ctx, client, je.tracker.Id, bugTodoSourceHutID, opr.Message)
				if err != nil {
					err := errors.Wrap(err, "editing issue body")
					out <- core.NewExportError(err, b.Id())
					return err
				}
				todosrhtOpID = fmt.Sprintf("%d", ticket.Id)
				exportTime = ticket.Updated.Unix()
				out <- core.NewExportCommentEdition(b.Id())
			} else {
				// For now, ignore comment edits that are not on the initial ticket body.
				// This is a limitation due to SourceHut API not directly supporting
				// editing arbitrary comments, but rather superseding them.
				out <- core.NewExportNothing(b.Id(), fmt.Sprintf("ignoring comment edit for op %s, SourceHut API limitation", op.Id().Human()))
				continue
			}

		case *bug.SetStatusOperation:
			// Map git-bug status to SourceHut TicketStatus and Resolution
			newStatus, newResolution, err := mapGitBugStatusToTodoSRHTStatusAndResolution(opr.Status)
			if err != nil {
				out <- core.NewExportError(errors.Wrap(err, "mapping status"), b.Id())
				return err
			}

			statusEvent, err := updateTodoSRHTTicketStatus(ctx, client, je.tracker.Id, bugTodoSourceHutID, newStatus, newResolution)
			if err != nil {
				err := errors.Wrap(err, "editing status")
				out <- core.NewExportError(err, b.Id())
				return err
			}
			todosrhtOpID = fmt.Sprintf("%d", statusEvent.Id)
			exportTime = statusEvent.Created.Unix()
			out <- core.NewExportStatusChange(b.Id())

		case *bug.SetTitleOperation:
			ticket, err := updateTodoSRHTTicketTitle(ctx, client, je.tracker.Id, bugTodoSourceHutID, opr.Title)
			if err != nil {
				err := errors.Wrap(err, "editing title")
				out <- core.NewExportError(err, b.Id())
				return err
			}
			todosrhtOpID = fmt.Sprintf("%d", ticket.Id)
			exportTime = ticket.Updated.Unix()
			out <- core.NewExportTitleEdition(b.Id())

		case *bug.LabelChangeOperation:
			// Fetch all labels for the tracker to get their IDs
			allLabels, err := je.getAllTrackerLabels(ctx, client)
			if err != nil {
				out <- core.NewExportError(errors.Wrap(err, "fetching all labels"), b.Id())
				return err
			}

			// Add labels
			for _, addedLabel := range opr.Added {
				labelID, found := findLabelIDByName(allLabels, string(addedLabel))
				if !found {
					// Create label if it doesn't exist
					// SourceHut GraphQL API for creating labels is a mutation that returns a Label
					// This would require a new mutation call if we want to create it on the fly.
					// For now, we will report a warning if label not found.
					out <- core.NewExportWarning(fmt.Errorf("label '%s' not found on SourceHut, cannot add", addedLabel), b.Id())
					continue
				}
				labelEvent, err := addTodoSRHTLabel(ctx, client, je.tracker.Id, bugTodoSourceHutID, labelID)
				if err != nil {
					out <- core.NewExportError(errors.Wrap(err, fmt.Sprintf("adding label '%s'", addedLabel)), b.Id())
					return err
				}
				// Use the last label event ID and time
				todosrhtOpID = fmt.Sprintf("%d", labelEvent.Id)
				exportTime = labelEvent.Created.Unix()
			}

			// Remove labels
			for _, removedLabel := range opr.Removed {
				labelID, found := findLabelIDByName(allLabels, string(removedLabel))
				if !found {
					out <- core.NewExportWarning(fmt.Errorf("label '%s' not found on SourceHut, cannot remove", removedLabel), b.Id())
					continue
				}
				labelEvent, err := removeTodoSRHTLabel(ctx, client, je.tracker.Id, bugTodoSourceHutID, labelID)
				if err != nil {
					out <- core.NewExportError(errors.Wrap(err, fmt.Sprintf("removing label '%s'", removedLabel)), b.Id())
					return err
				}
				// Use the last label event ID and time
				todosrhtOpID = fmt.Sprintf("%d", labelEvent.Id)
				exportTime = labelEvent.Created.Unix()
			}
			out <- core.NewExportLabelChange(b.Id())

		default:
			panic("unhandled operation type case")
		}
		// mark operation as exported
		err = markOperationAsExported(
			b, op.Id(), todosrhtOpID, je.tracker.Name, exportTime)
		if err != nil {
			err := errors.Wrap(err, "marking operation as exported")
			out <- core.NewExportError(err, b.Id())
			return err
		}

		// commit at each operation export to avoid exporting same events multiple
		// times
		err = b.CommitAsNeeded()
		if err != nil {
			err := errors.Wrap(err, "bug commit")
			out <- core.NewExportError(err, b.Id())
			return err
		}
	}

	return nil
}

// markOperationAsExported adapted for SourceHut metadata
func markOperationAsExported(b *cache.BugCache, targetOpID entity.Id, todosrhtID string, todosrhtTrackerName string, exportTime int64) error {
	newMetadata := map[string]string{
		metaKeyTodoSourceHutId:      todosrhtID,
		metaKeyTodoSourceHutTracker: todosrhtTrackerName,
	}
	if exportTime != 0 {
		newMetadata[metaKeyTodoSourceHutExportTime] = time.Unix(exportTime, 0).Format(time.RFC3339)
	}

	_, err := b.SetMetadata(targetOpID, newMetadata)
	return err
}

func createTodoSRHTTicket(ctx context.Context, client TodosrhtClient, trackerID int, subject, body string) (*Ticket, error) {
	input := SubmitTicketInput{
		Subject: subject,
		Body:    body,
	}
	return client.CreateTicket(ctx, trackerID, input)
}

func addTodoSRHTComment(ctx context.Context, client TodosrhtClient, trackerID, ticketID int, text string) (*Event, error) {
	input := SubmitCommentInput{
		Text: text,
		// SourceHut's SubmitCommentInput has Status and Resolution fields, but they
		// are typically used for specific status changes related to the comment.
		// For a simple comment, we leave them as default/unspecified.
	}
	return client.CreateComment(ctx, trackerID, ticketID, input)
}

func updateTodoSRHTTicketBody(ctx context.Context, client TodosrhtClient, trackerID, ticketID int, body string) (*Ticket, error) {
	input := UpdateTicketInput{
		Body: body,
	}
	return client.UpdateTicket(ctx, trackerID, ticketID, input)
}

func updateTodoSRHTTicketTitle(ctx context.Context, client TodosrhtClient, trackerID, ticketID int, title string) (*Ticket, error) {
	input := UpdateTicketInput{
		Subject: title,
	}
	return client.UpdateTicket(ctx, trackerID, ticketID, input)
}

func updateTodoSRHTTicketStatus(ctx context.Context, client TodosrhtClient, trackerID, ticketID int, status TicketStatus, resolution TicketResolution) (*Event, error) {
	input := UpdateStatusInput{
		Status:     status,
		Resolution: resolution,
	}
	return client.UpdateTicketStatus(ctx, trackerID, ticketID, input)
}

func addTodoSRHTLabel(ctx context.Context, client TodosrhtClient, trackerID, ticketID, labelID int) (*Event, error) {
	return client.AddLabel(ctx, trackerID, ticketID, labelID)
}

func removeTodoSRHTLabel(ctx context.Context, client TodosrhtClient, trackerID, ticketID, labelID int) (*Event, error) {
	return client.RemoveLabel(ctx, trackerID, ticketID, labelID)
}

func (je *todosrhtExporter) getAllTrackerLabels(ctx context.Context, client TodosrhtClient) ([]Label, error) {
	// je.cachedLabels is a map[string]string (name -> ID) so it's not storing Label objects.
	// For now, let's just refetch all labels every time.
	// A proper cache for Label objects could be implemented later.

	var allLabels []Label
	var cursor *string
	for {
		labelsCursor, err := client.GetLabels(ctx, je.tracker.Id, cursor)
		if err != nil {
			return nil, err
		}
		allLabels = append(allLabels, labelsCursor.Results...)
		if labelsCursor.Cursor == nil || *labelsCursor.Cursor == "" {
			break
		}
		cursor = labelsCursor.Cursor
	}
	return allLabels, nil
}

func findLabelIDByName(labels []Label, name string) (int, bool) {
	for _, label := range labels {
		if strings.EqualFold(label.Name, name) {
			return label.Id, true
		}
	}
	return 0, false
}

func mapGitBugStatusToTodoSRHTStatusAndResolution(s common.Status) (TicketStatus, TicketResolution, error) {
	switch s {
	case common.OpenStatus:
		return TicketStatusReported, TicketResolutionUnresolved, nil // Default to REPORTED and UNRESOLVED for open
	case common.ClosedStatus:
		return TicketStatusResolved, TicketResolutionFixed, nil // Default to RESOLVED and FIXED for closed
	default:
		return "", "", fmt.Errorf("unknown git-bug status: %s", s)
	}
}
