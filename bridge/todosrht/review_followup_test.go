package todosrht

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/entity"
)

func TestBodyWhitespaceDoesNotBlockPull(t *testing.T) {
	for _, baseline := range []string{"exported", "raw-checkpoint", "legacy-mapping"} {
		t.Run(baseline, func(t *testing.T) {
			backend, user := reviewBackend(t)
			b := mappedReviewBug(t, backend)
			_, edit, err := b.EditCreateComment("sent\n")
			require.NoError(t, err)
			require.NoError(t, b.CommitAsNeeded())
			if baseline == "legacy-mapping" {
				require.NoError(t, markOperationAsExported(b, edit.Id(), "1", "~tester/issues", 1001))
				require.NoError(t, b.CommitAsNeeded())
			} else {
				client := &MockClient{MockUpdateTicket: func(_ context.Context, tracker, id int, input UpdateTicketInput) (*Ticket, error) {
					require.NotNil(t, input.Body)
					require.Equal(t, "sent\n", *input.Body)
					return &Ticket{Id: id, Body: *input.Body}, nil
				}}
				exporter := &todosrhtExporter{conf: reviewImporter("tester").conf,
					tracker: trackerForTest("tester", "issues"), cachedOperationIDs: make(map[entity.Id]string),
					identityClient: map[entity.Id]TodosrhtClient{user.Id(): client},
				}
				require.NoError(t, exporter.exportBug(t.Context(), b, make(chan core.ExportResult, 100)))
				if baseline == "raw-checkpoint" {
					// Simulate a checkpoint written before normalization was added.
					raw, err := json.Marshal(remoteState{Subject: "Ticket", Body: "sent\n"})
					require.NoError(t, err)
					ops := b.Snapshot().Operations
					_, err = b.SetMetadata(ops[len(ops)-1].Id(), map[string]string{metaKeyRemoteState: string(raw)})
					require.NoError(t, err)
					require.NoError(t, b.CommitAsNeeded())
				}
			}
			_, _, err = b.EditCreateComment("pending local body\n")
			require.NoError(t, err)
			require.NoError(t, b.CommitAsNeeded())
			ticket := reviewTicket()
			ticket.Subject, ticket.Body = "Ticket", "sent\n"
			comment := json.RawMessage(`{"__typename":"Comment","author":{"__typename":"User","canonicalName":"~tester","username":"tester"},"text":"New remote comment"}`)
			importer := reviewImporter("tester")
			importer.client = &MockClient{
				MockGetTracker: func(context.Context, string) (*Tracker, error) { return trackerForTest("tester", "issues"), nil },
				MockGetTickets: func(context.Context, string, *string) ([]Ticket, *string, error) { return []Ticket{ticket}, nil, nil },
				MockGetEvents: func(context.Context, string, int, *string) ([]Event, *string, error) {
					return []Event{{Id: 42, Created: Time(time.Unix(2001, 0)), Changes: []json.RawMessage{comment}}}, nil, nil
				},
			}
			for pass := 0; pass < 2; pass++ {
				results, err := importer.ImportAll(t.Context(), backend, time.Time{})
				require.NoError(t, err)
				for result := range results {
					require.NoError(t, result.Err)
				}
				b, err = backend.Bugs().Resolve(b.Id())
				require.NoError(t, err)
				require.Equal(t, "pending local body\n", b.Snapshot().Comments[0].Message)
				require.Len(t, b.Snapshot().Comments, 2)
				require.Equal(t, "New remote comment", b.Snapshot().Comments[1].Message)
			}
		})
	}
}

func TestWhitespaceOnlyLocalEditDoesNotConflict(t *testing.T) {
	backend, _ := reviewBackend(t)
	importer := reviewImporter("tester")
	ticket := reviewTicket()
	b, _, err := importer.ensureIssue(backend, ticket)
	require.NoError(t, err)
	_, _, err = b.EditCreateComment(ticket.Body + "\n")
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())
	ticket.Body = "Remote body edit\n"
	b, _, err = importer.ensureIssue(backend, ticket)
	require.NoError(t, err)
	require.Equal(t, "Remote body edit", b.Snapshot().Comments[0].Message)
}

func TestUnsupportedCommentEditNeedsNoCredentials(t *testing.T) {
	backend, user := reviewBackend(t)
	other, err := backend.Identities().New("Other", "other@example.org")
	require.NoError(t, err)
	b := mappedReviewBug(t, backend)
	comment, op, err := b.AddComment("Existing exported comment")
	require.NoError(t, err)
	require.NoError(t, markOperationAsExported(b, op.Id(), "100", "~tester/issues", 1000))
	_, err = b.EditCommentRaw(other, 1001, comment, "Unsupported comment edit", nil)
	require.NoError(t, err)
	_, err = b.CloseRaw(user, 1002, nil)
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())
	_, _, err = backend.Bugs().New("Another bug", "Body")
	require.NoError(t, err)
	var statusChanges, createdTickets int
	client := &MockClient{
		MockUpdateTicketStatus: func(context.Context, int, int, UpdateStatusInput) (*Event, error) {
			statusChanges++
			return &Event{Id: 101}, nil
		},
		MockCreateTicket: func(context.Context, int, SubmitTicketInput) (*Ticket, error) {
			createdTickets++
			return &Ticket{Id: 2}, nil
		},
	}
	exporter := &todosrhtExporter{conf: reviewImporter("tester").conf,
		tracker: trackerForTest("tester", "issues"), cachedOperationIDs: make(map[entity.Id]string),
		identityClient: map[entity.Id]TodosrhtClient{user.Id(): client},
	}
	for pass := 0; pass < 2; pass++ {
		results, err := exporter.ExportAll(t.Context(), backend, time.Time{})
		require.NoError(t, err)
		for result := range results {
			require.NoError(t, result.Err)
		}
		require.Equal(t, 1, statusChanges)
		require.Equal(t, 1, createdTickets)
	}

	// A supported body edit by the same uncredentialed author must still
	// block the later reopen, rather than sending supported changes out of order.
	b, err = backend.Bugs().Resolve(b.Id())
	require.NoError(t, err)
	_, _, err = b.EditCreateCommentRaw(other, 1003, "Supported body edit", nil)
	require.NoError(t, err)
	_, err = b.OpenRaw(user, 1004, nil)
	require.NoError(t, err)
	require.NoError(t, b.CommitAsNeeded())
	results, err := exporter.ExportAll(t.Context(), backend, time.Time{})
	require.NoError(t, err)
	var blocked bool
	for result := range results {
		if result.Err != nil {
			blocked = true
		}
	}
	require.True(t, blocked)
	require.Equal(t, 1, statusChanges)
	require.Equal(t, 1, createdTickets)
}
