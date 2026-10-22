package todosrht

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/git-bug/git-bug/bridge/core"
	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/entity"
)

func TestMultiLabelExportRecordsEveryEventAcrossRetry(t *testing.T) {
	backend, user := reviewBackend(t)
	b := mappedReviewBug(t, backend)
	_, _, err := b.ChangeLabels(t.Context(), []string{"a", "b"}, nil)
	require.NoError(t, err)
	_, _, err = b.ChangeLabels(t.Context(), nil, []string{"a"})
	require.NoError(t, err)
	require.NoError(t, b.Commit(t.Context()))
	present := map[int]bool{}
	fail := true
	var events []Event
	makeEvent := func(label int, added bool) *Event {
		kind, name := EventTypeLabelRemoved, "a"
		if added {
			kind = EventTypeLabelAdded
		}
		if label == 2 {
			name = "b"
		}
		change := json.RawMessage(fmt.Sprintf(`{"__typename":"LabelUpdate","eventType":%q,"labeler":{"__typename":"User","canonicalName":"~tester","username":"tester"},"label":{"id":%d,"name":%q}}`, kind, label, name))
		event := Event{Id: 10 + len(events), Created: Time(time.Unix(int64(1010+len(events)), 0)), Changes: []json.RawMessage{change}}
		events = append(events, event)
		return &event
	}
	client := &MockClient{
		MockGetLabels: func(context.Context, string, *string) (*LabelCursor, error) {
			return &LabelCursor{Results: []Label{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}}, nil
		},
		MockGetTicket: func(context.Context, string, int) (*Ticket, error) {
			ticket := &Ticket{Id: 1}
			for id := range present {
				ticket.Labels = append(ticket.Labels, Label{Id: id})
			}
			return ticket, nil
		},
		MockAddLabel: func(_ context.Context, _, _, id int) (*Event, error) {
			if id == 2 && fail {
				return nil, fmt.Errorf("temporary label failure")
			}
			present[id] = true
			return makeEvent(id, true), nil
		},
		MockRemoveLabel: func(_ context.Context, _, _, id int) (*Event, error) {
			delete(present, id)
			return makeEvent(id, false), nil
		},
	}
	exporter := &todosrhtExporter{conf: reviewImporter("tester").conf,
		tracker: trackerForTest("tester", "issues"), cachedOperationIDs: map[entity.Id]string{},
		identityClient: map[entity.Id]TodosrhtClient{user.Id(): client}}
	require.Error(t, exporter.exportBug(t.Context(), b, make(chan core.ExportResult, 100)))
	fail = false
	b, err = backend.Bugs().Resolve(t.Context(), b.Id())
	require.NoError(t, err)
	require.NoError(t, exporter.exportBug(t.Context(), b, make(chan core.ExportResult, 100)))
	require.Len(t, events, 3)
	b, err = backend.Bugs().Resolve(t.Context(), b.Id())
	require.NoError(t, err)
	for _, event := range events {
		require.NoError(t, reviewImporter("tester").ensureEvent(t.Context(), backend, b, event))
	}
	require.Equal(t, []common.Label{"b"}, b.Snapshot().Labels)
}

func TestAcceptedRemoteEditsDoNotExportSupersededLocalEdits(t *testing.T) {
	for _, field := range []string{"body-whitespace", "body-revert", "title-revert"} {
		t.Run(field, func(t *testing.T) {
			backend, user := reviewBackend(t)
			b := mappedReviewBug(t, backend)
			var err error
			switch field {
			case "body-whitespace":
				_, _, err = b.EditCreateComment(t.Context(), "Body\n")
			case "body-revert":
				_, _, err = b.EditCreateComment(t.Context(), "Temporary")
				require.NoError(t, err)
				_, _, err = b.EditCreateComment(t.Context(), "Body")
			case "title-revert":
				_, err = b.SetTitle(t.Context(), "Temporary")
				require.NoError(t, err)
				_, err = b.SetTitle(t.Context(), "Ticket")
			}
			require.NoError(t, err)
			require.NoError(t, b.Commit(t.Context()))
			ticket := reviewTicket()
			ticket.Subject, ticket.Body = "Ticket", "Remote body"
			if field == "title-revert" {
				ticket.Subject, ticket.Body = "Remote title", "Body"
			}
			b, _, err = reviewImporter("tester").ensureIssue(t.Context(), backend, ticket)
			require.NoError(t, err)
			require.NoError(t, b.Commit(t.Context()))
			requests := 0
			client := &MockClient{MockUpdateTicket: func(context.Context, int, int, UpdateTicketInput) (*Ticket, error) {
				requests++
				return &Ticket{Id: 1}, nil
			}}
			exporter := &todosrhtExporter{conf: reviewImporter("tester").conf,
				tracker: trackerForTest("tester", "issues"), cachedOperationIDs: map[entity.Id]string{},
				identityClient: map[entity.Id]TodosrhtClient{user.Id(): client}}
			require.NoError(t, exporter.exportBug(t.Context(), b, make(chan core.ExportResult, 100)))
			require.Zero(t, requests)
		})
	}
}

func TestUnsupportedEventsNeedNoCurrentUser(t *testing.T) {
	backend, _ := reviewBackend(t)
	b := mappedReviewBug(t, backend)
	require.NoError(t, backend.ClearUserIdentity())
	importer := reviewImporter("tester")
	warning := json.RawMessage(`{"__typename":"Assignment","eventType":"ASSIGNED_USER"}`)
	status := json.RawMessage(`{"__typename":"StatusChange","eventType":"STATUS_CHANGE","editor":{"__typename":"User","canonicalName":"~tester","username":"tester"},"newStatus":"RESOLVED"}`)
	for pass := 0; pass < 2; pass++ {
		require.NoError(t, importer.ensureEvent(t.Context(), backend, b, Event{Id: 5, Changes: []json.RawMessage{warning}}))
		require.NoError(t, importer.ensureEvent(t.Context(), backend, b, Event{Id: 6, Changes: []json.RawMessage{status}}))
		require.NoError(t, b.CommitAsNeeded(t.Context()))
		var err error
		b, err = backend.Bugs().Resolve(t.Context(), b.Id())
		require.NoError(t, err)
		require.Equal(t, common.ClosedStatus, b.Snapshot().Status)
	}
}

func TestSameOriginRedirectLoopIsBounded(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Redirect(w, r, "/query", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	_, err := NewTodoSClient(t.Context(), server.URL, "token").GetTracker(t.Context(), "issues")
	require.ErrorContains(t, err, "10 redirects")
	require.Equal(t, 10, requests)
}
