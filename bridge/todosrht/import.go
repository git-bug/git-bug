package todosrht

import (
	"context"
	"fmt"
	"time"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/bridge/core/auth"
	"github.com/git-bug/git-bug/cache"
	"github.com/git-bug/git-bug/entities/bug"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
	"github.com/git-bug/git-bug/util/text"
)

// todosrhtImporter implement the Importer interface
type todosrhtImporter struct {
	conf core.Configuration

	client TodosrhtClient

	// send only channel
	out chan<- core.ImportResult
}

// Init .
func (ji *todosrhtImporter) Init(_ context.Context, repo *cache.RepoCache, conf core.Configuration) error {
	ji.conf = conf

	creds, err := auth.List(repo,
		auth.WithTarget(target),
		auth.WithKind(auth.KindToken),
		auth.WithMeta(auth.MetaKeyBaseURL, conf[confKeyBaseUrl]),
		auth.WithMeta(auth.MetaKeyLogin, conf[confKeyDefaultLogin]),
	)
	if err != nil {
		return err
	}

	if len(creds) == 0 {
		return ErrMissingCredentials
	}

	tokenCred, ok := creds[0].(*auth.Token)
	if !ok {
		return fmt.Errorf("expected token credential, got %T", creds[0])
	}

	ji.client = NewTodoSClient(context.TODO(), conf[confKeyBaseUrl], tokenCred.Value)

	return nil
}

// ImportAll iterate over all the configured repository issues and ensure the
// creation of the missing issues / timeline items / edits / label events ...
func (ji *todosrhtImporter) ImportAll(ctx context.Context, repo *cache.RepoCache, since time.Time) (<-chan core.ImportResult, error) {
	out := make(chan core.ImportResult)
	ji.out = out

	go func() {
		defer close(ji.out)

		trackerName := ji.conf[confKeyTrackerName]
		tracker, err := ji.client.GetTracker(ctx, trackerName)
		if err != nil {
			ji.out <- core.NewImportError(fmt.Errorf("failed to get tracker: %w", err), "")
			return
		}

		if tracker == nil {
			ji.out <- core.NewImportError(fmt.Errorf("tracker '%s' not found", trackerName), "")
			return
		}

		var cursor *string
		for {
			tickets, nextCursor, err := ji.client.GetTickets(ctx, tracker.Name, cursor)
			if err != nil {
				ji.out <- core.NewImportError(fmt.Errorf("failed to get tickets: %w", err), "")
				return
			}

			for _, ticket := range tickets {
				// Filter by since time
				if ticket.Updated.Unix() < since.Unix() {
					ji.out <- core.NewImportNothing(entity.Id(fmt.Sprintf("%d", ticket.Id)), "ticket updated before since date")
					continue
				}

				b, err := ji.ensureIssue(repo, ticket)
				if err != nil {
					ji.out <- core.NewImportError(fmt.Errorf("failed to ensure issue for ticket %d: %w", ticket.Id, err), "")
					continue
				}

				var eventCursor *string
				for {
					events, nextEventCursor, err := ji.client.GetEvents(ctx, ticket.Id, eventCursor)
					if err != nil {
						ji.out <- core.NewImportError(fmt.Errorf("failed to get events for ticket %d: %w", ticket.Id, err), b.Id())
						break
					}

					for _, event := range events {
						if err := ji.ensureEvent(repo, b, event); err != nil {
							ji.out <- core.NewImportError(fmt.Errorf("failed to ensure event %d for ticket %d: %w", event.Id, ticket.Id, err), b.Id())
							continue
						}
					}

					if nextEventCursor == nil || *nextEventCursor == "" {
						break
					}
					eventCursor = nextEventCursor
				}

				if b.NeedCommit() {
					if err := b.Commit(); err != nil {
						ji.out <- core.NewImportError(fmt.Errorf("failed to commit bug for ticket %d: %w", ticket.Id, err), b.Id())
					}
				} else {
					ji.out <- core.NewImportNothing(b.Id(), "no new operations imported")
				}
			}

			if nextCursor == nil || *nextCursor == "" {
				break
			}
			cursor = nextCursor
		}
	}()

	return out, nil
}

// Create a bug.Person from a TODOSRHT user
func (ji *todosrhtImporter) ensurePerson(repo *cache.RepoCache, entities Entity) (*cache.IdentityCache, error) {
	var canonicalName, username, email, externalId, externalUrl string

	// Determine the concrete type of the entity
	switch e := entities.(type) {
	case User:
		canonicalName = e.CanonicalName
		username = e.Username
		email = e.Email
	case ExternalUser:
		canonicalName = e.CanonicalName
		externalId = e.ExternalId
		externalUrl = e.ExternalUrl
	default:
		return nil, fmt.Errorf("unknown entity type %T", entities)
	}

	// Look first in the cache
	i, err := repo.Identities().ResolveIdentityImmutableMetadata(
		metaKeyTodoSourceHutLogin, canonicalName)
	if err == nil {
		return i, nil
	}
	if _, ok := err.(entity.ErrMultipleMatch); ok {
		return nil, err
	}

	// If not found, create a new identity
	metadata := map[string]string{
		metaKeyTodoSourceHutLogin: canonicalName,
	}
	if externalId != "" {
		metadata[metaKeyTodoSourceHutUser] = externalId
	}
	if externalUrl != "" {
		metadata["url"] = externalUrl // Store external URL if available
	}

	i, err = repo.Identities().NewRaw(
		// Use username or canonical name for display, email for email
		func() string {
			if username != "" {
				return username
			}
			return canonicalName
		}(),
		email, // Can be empty for ExternalUser
		canonicalName,
		"", // Avatar URL is not directly available in SourceHut API for now
		nil,
		metadata,
	)

	if err != nil {
		return nil, err
	}

	ji.out <- core.NewImportIdentity(i.Id())
	return i, nil
}

// Create a bug.Bug based from a TODOSRHT ticket
func (ji *todosrhtImporter) ensureIssue(repo *cache.RepoCache, ticket Ticket) (*cache.BugCache, error) {
	submitter, err := ticket.GetSubmitter()
	if err != nil {
		ji.out <- core.NewImportError(fmt.Errorf("failed to parse submitter for ticket %d: %w", ticket.Id, err), "")
		return nil, err
	}
	author, err := ji.ensurePerson(repo, submitter)
	if err != nil {
		return nil, err
	}

	b, err := repo.Bugs().ResolveMatcher(func(excerpt *cache.BugExcerpt) bool {
		if _, ok := excerpt.CreateMetadata[metaKeyTodoSourceHutBaseUrl]; ok &&
			excerpt.CreateMetadata[metaKeyTodoSourceHutBaseUrl] != ji.conf[confKeyBaseUrl] {
			return false
		}

		// Use the tracker name and ticket ID to uniquely identify the bug
		return excerpt.CreateMetadata[core.MetaKeyOrigin] == target &&
			excerpt.CreateMetadata[metaKeyTodoSourceHutTracker] == ji.conf[confKeyTrackerName] &&
			excerpt.CreateMetadata[metaKeyTodoSourceHutId] == fmt.Sprintf("%d", ticket.Id)
	})
	if err != nil && !entity.IsErrNotFound(err) {
		return nil, err
	}

	if entity.IsErrNotFound(err) {
		// New ticket, create a new bug
		b, _, err = repo.Bugs().NewRaw(
			author,
			ticket.Created.Unix(),
			text.CleanupOneLine(ticket.Subject),
			text.Cleanup(ticket.Body),
			nil,
			map[string]string{
				core.MetaKeyOrigin:          target,
				metaKeyTodoSourceHutId:      fmt.Sprintf("%d", ticket.Id),
				metaKeyTodoSourceHutTracker: ji.conf[confKeyTrackerName],
				metaKeyTodoSourceHutRef:     ticket.Ref, // Store ticket reference
				metaKeyTodoSourceHutBaseUrl: ji.conf[confKeyBaseUrl],
			})
		if err != nil {
			return nil, err
		}

		ji.out <- core.NewImportBug(b.Id())
	}

	return b, nil
}

// ensureEvent processes a SourceHut event and creates corresponding git-bug operations.
func (ji *todosrhtImporter) ensureEvent(repo *cache.RepoCache, b *cache.BugCache, event Event) error {
	// Check if this event has already been imported
	_, err := b.ResolveOperationWithMetadata(metaKeyTodoSourceHutId, fmt.Sprintf("%d", event.Id))
	if err == nil {
		return nil // Already imported
	}
	if err != cache.ErrNoMatchingOp {
		return err // Real error
	}

	for _, change := range event.Changes {
		switch c := change.(type) {
		case Created:
			// The Created event is handled by ensureIssue when creating the bug itself.
			// We only need to ensure the author is registered if not already.
			_, err := ji.ensurePerson(repo, c.Author)
			if err != nil {
				return err
			}
			// Mark this event ID as processed
			_, err = b.SetMetadata(b.Snapshot().Operations[0].Id(), map[string]string{
				metaKeyTodoSourceHutId: fmt.Sprintf("%d", event.Id),
			})
			if err != nil {
				return err
			}

		case Comment:
			author, err := ji.ensurePerson(repo, c.Author)
			if err != nil {
				return err
			}

			// Add comment operation
			commentId, _, err := b.AddCommentRaw(
				author,
				event.Created.Unix(),
				text.Cleanup(c.Text),
				nil,
				map[string]string{
					metaKeyTodoSourceHutId: fmt.Sprintf("%d", event.Id),
				},
			)
			if err != nil {
				return err
			}
			ji.out <- core.NewImportComment(b.Id(), commentId)

		case StatusChange:
			editor, err := ji.ensurePerson(repo, c.Editor)
			if err != nil {
				return err
			}

			var op *bug.SetStatusOperation
			if c.NewStatus == TicketStatusResolved {
				op, err = b.CloseRaw(
					editor,
					event.Created.Unix(),
					map[string]string{
						metaKeyTodoSourceHutId: fmt.Sprintf("%d", event.Id),
					},
				)
			} else {
				// Assuming all other statuses map to Open, or require more granular mapping
				op, err = b.OpenRaw(
					editor,
					event.Created.Unix(),
					map[string]string{
						metaKeyTodoSourceHutId: fmt.Sprintf("%d", event.Id),
					},
				)
			}
			if err != nil {
				return err
			}
			ji.out <- core.NewImportStatusChange(b.Id(), op.Id())

		case LabelUpdate:
			labeler, err := ji.ensurePerson(repo, c.Labeler)
			if err != nil {
				return err
			}

			var added, removed []common.Label
			if c.EventTypeVal == EventTypeLabelAdded {
				added = []common.Label{common.Label(c.Label.Name)}
			} else if c.EventTypeVal == EventTypeLabelRemoved {
				removed = []common.Label{common.Label(c.Label.Name)}
			}

			op, err := b.ForceChangeLabelsRaw(
				labeler,
				event.Created.Unix(),
				labelsToStrings(added),
				labelsToStrings(removed),
				map[string]string{
					metaKeyTodoSourceHutId: fmt.Sprintf("%d", event.Id),
				},
			)
			if err != nil {
				return err
			}
			ji.out <- core.NewImportLabelChange(b.Id(), op.Id())

		// Add other event types as needed (Assignment, UserMention, TicketMention)
		// For now, we'll ignore them or log a warning if they are not directly
		// mappable to git-bug operations or are not high priority.
		default:
			ji.out <- core.NewImportWarning(
				fmt.Errorf("unhandled SourceHut event detail type: %T for event %d", c, event.Id),
				b.Id(),
			)
		}
	}
	return nil
}

func labelsToStrings(labels []common.Label) []string {
	res := make([]string, len(labels))
	for i, l := range labels {
		res[i] = string(l)
	}
	return res
}
